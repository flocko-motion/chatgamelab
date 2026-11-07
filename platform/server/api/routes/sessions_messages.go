// package: routes / game session HTTP handlers
// type:    logic
// job:     HTTP handlers for session message status, image, audio, and streaming.
// limits:  does not manage sessions themselves (-> sessions.go).
package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/game/imagecache"
	"cgl/game/stream"
	"cgl/log"
	"cgl/obj"
)

// MessageStatusResponse is the unified response for polling message completion.
// Frontend polls this to catch up after SSE drops, on reload, or for image progress.
type MessageStatusResponse struct {
	Text         string            `json:"text"`                   // Current full text of the message
	TextDone     bool              `json:"textDone"`               // True when text streaming is complete (Stream=false in DB)
	ImageStatus  string            `json:"imageStatus"`            // "none" | "generating" | "complete" | "error"
	ImageHash    string            `json:"imageHash,omitempty"`    // Hash for cache-busting image URL
	ImageError   string            `json:"imageError,omitempty"`   // Machine-readable image error code
	StatusFields []obj.StatusField `json:"statusFields,omitempty"` // Current status fields
	Error        string            `json:"error,omitempty"`        // Fatal error message (AI failure)
	ErrorCode    string            `json:"errorCode,omitempty"`    // Machine-readable error code
}

// GetMessageStatus godoc
//
//	@Summary		Get message completion status
//	@Description	Returns the current state of a message: text, image status, errors.
//	@Description	Frontend polls this as a safety net when SSE drops, on reload, or for image progress.
//	@Description	No authentication required - message UUIDs are random and unguessable.
//	@Tags			messages
//	@Produce		json
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{object}	MessageStatusResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message not found"
//	@Router			/messages/{id}/status [get]
func GetMessageStatus(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Get message from DB (no auth - relies on UUID unguessability, same as image endpoint)
	msg, err := db.GetGameSessionMessageByIDPublic(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	// Determine text completion: if stream registry has an active stream, text is still in progress
	registry := stream.Get()
	textStreaming := registry.Lookup(messageID) != nil

	resp := MessageStatusResponse{
		Text:         msg.Message,
		TextDone:     !textStreaming,
		StatusFields: msg.StatusFields,
	}

	// Determine image status
	if msg.ImagePrompt == nil || *msg.ImagePrompt == "" {
		resp.ImageStatus = "none"
	} else {
		// Check image cache first (in-progress generation)
		cache := imagecache.Get()
		imgStatus := cache.GetStatus(messageID)

		if imgStatus.Exists {
			if imgStatus.HasError {
				resp.ImageStatus = "error"
				resp.ImageError = imgStatus.ErrorCode
			} else if imgStatus.IsComplete {
				resp.ImageStatus = "complete"
				resp.ImageHash = imgStatus.Hash
			} else {
				resp.ImageStatus = "generating"
				resp.ImageHash = imgStatus.Hash
			}
		} else if len(msg.Image) > 0 {
			// Image already persisted to DB
			resp.ImageStatus = "complete"
			resp.ImageHash = "persisted"
		} else if textStreaming {
			// Stream still active, image generation hasn't started yet
			resp.ImageStatus = "generating"
		} else {
			// Stream finished but no image - generation failed silently or was skipped
			resp.ImageStatus = "none"
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

// ImageStatusResponse is the response for the image status endpoint
type ImageStatusResponse struct {
	Hash                     string `json:"hash"`
	IsComplete               bool   `json:"isComplete"`
	HasError                 bool   `json:"hasError,omitempty"`
	ErrorCode                string `json:"errorCode,omitempty"`
	ErrorMsg                 string `json:"errorMsg,omitempty"`
	Exists                   bool   `json:"exists"`
	IsOrganisationUnverified bool   `json:"isOrganisationUnverified,omitempty"`
}

// GetMessageImageStatus godoc
//
//	@Summary		Get image generation status
//	@Description	Returns the current hash and completion status of an image being generated.
//	@Description	Frontend can poll this to detect when new partial/final images are available.
//	@Tags			messages
//	@Produce		json
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{object}	ImageStatusResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Router			/messages/{id}/image/status [get]
func GetMessageImageStatus(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Check cache first for in-progress images
	cache := imagecache.Get()
	status := cache.GetStatus(messageID)

	if status.Exists {
		resp := ImageStatusResponse{
			Hash:       status.Hash,
			IsComplete: status.IsComplete,
			HasError:   status.HasError,
			ErrorCode:  status.ErrorCode,
			ErrorMsg:   status.ErrorMsg,
			Exists:     true,
		}

		// If there's an org verification error, also set the flag
		if status.ErrorCode == obj.ErrCodeOrgVerificationRequired {
			resp.IsOrganisationUnverified = true
		}

		httpx.WriteJSON(w, http.StatusOK, resp)
		return
	}

	// Check if image exists in DB (already completed)
	msg, err := db.GetGameSessionMessageImageByID(r.Context(), messageID)
	if err == nil && len(msg.Image) > 0 {
		httpx.WriteJSON(w, http.StatusOK, ImageStatusResponse{
			Hash:       "persisted",
			IsComplete: true,
			Exists:     true,
		})
		return
	}

	// No image in cache or DB
	httpx.WriteJSON(w, http.StatusOK, ImageStatusResponse{
		Exists: false,
	})
}

// GetMessageImage godoc
//
//	@Summary		Get message image
//	@Description	Returns the generated image for a game session message.
//	@Description	Checks in-memory cache first (for partial/WIP images), then database.
//	@Description	No authentication required - message UUIDs are random and unguessable.
//	@Tags			messages
//	@Produce		image/png
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{file}		binary	"PNG image"
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message or image not found"
//	@Router			/messages/{id}/image [get]
func GetMessageImage(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Check cache first for in-progress/partial images
	cache := imagecache.Get()
	if imageData, exists := cache.GetImage(messageID); exists {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache") // Don't cache partial images
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
		return
	}

	// Fall back to database for persisted images
	msg, err := db.GetGameSessionMessageImageByID(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	if len(msg.Image) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "Image not found")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	w.Write(msg.Image)
}

// GetMessageAudio godoc
//
//	@Summary		Get message audio
//	@Description	Returns the audio narration for a message (MP3 format).
//	@Description	No authentication required - message UUIDs are random and unguessable.
//	@Tags			messages
//	@Produce		audio/mpeg
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{file}		binary
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid message ID"
//	@Failure		404	{object}	httpx.ErrorResponse	"Message or audio not found"
//	@Router			/messages/{id}/audio [get]
func GetMessageAudio(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	audio, err := db.GetGameSessionMessageAudioByID(r.Context(), messageID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Message not found")
		return
	}

	if len(audio) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "Audio not found")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.WriteHeader(http.StatusOK)
	w.Write(audio)
}

// GetMessageStream godoc
//
//	@Summary		Stream message updates (SSE)
//	@Description	Server-Sent Events endpoint for streaming message chunks.
//	@Tags			messages
//	@Produce		text/event-stream
//	@Param			id	path		string	true	"Message ID (UUID)"
//	@Success		200	{string}	string	"SSE stream"
//	@Failure		400	{string}	string	"Invalid message ID"
//	@Failure		404	{string}	string	"Stream not found"
//	@Router			/messages/{id}/stream [get]
func GetMessageStream(w http.ResponseWriter, r *http.Request) {
	messageID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// Lookup the stream
	registry := stream.Get()
	s := registry.Lookup(messageID)
	if s == nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Use origin from request for CORS with credentials support
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream chunks to client until text, image, and audio are all done
	log.Debug("[SSE] client connected", "message_id", messageID, "buffered_chunks", len(s.Chunks), "closed", s.IsClosed())
	textDone := false
	imageDone := false
	audioDone := false
	chunkCount := 0

	// Detect client disconnect
	clientGone := r.Context().Done()

	for {
		select {
		case <-clientGone:
			log.Warn("[SSE] client disconnected", "message_id", messageID, "chunks_sent", chunkCount, "textDone", textDone, "imageDone", imageDone, "audioDone", audioDone)
			// Don't Remove the stream — producers are still writing.
			// The stream will be cleaned up when it completes normally or by the timeout.
			return
		case chunk, ok := <-s.Chunks:
			if !ok {
				log.Debug("[SSE] channel closed", "message_id", messageID, "chunks_sent", chunkCount)
				goto done
			}
			chunkCount++
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			if chunk.TextDone {
				textDone = true
			}
			if chunk.ImageDone {
				imageDone = true
			}
			if chunk.AudioDone {
				audioDone = true
			}
			if chunk.Error != "" {
				goto done
			}
			// Stream is complete when all active channels are done
			if textDone && imageDone && audioDone {
				goto done
			}
		}
	}

done:
	log.Debug("[SSE] stream completed", "message_id", messageID, "chunks", chunkCount, "textDone", textDone, "imageDone", imageDone, "audioDone", audioDone)

	// Cleanup
	registry.Remove(messageID)
}
