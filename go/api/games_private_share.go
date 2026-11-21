// package: api / schemaf endpoints
// type:    logic
// job:     creates, lists, updates and revokes private share links for games
// limits:  share storage lives in db; playing through a share link (-> guest_play.go)
package api

import (
	"context"
	"fmt"

	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// GameShareResp is a game share with its share URL.
type GameShareResp struct {
	obj.GameShare
	ShareURL string `json:"shareUrl"`
}

func toGameShareResp(gs obj.GameShare) GameShareResp {
	return GameShareResp{
		GameShare: gs,
		ShareURL:  "/play/" + gs.Token,
	}
}

// GameShareIDReq names a share of a game by the game's id and the share's id.
type GameShareIDReq struct {
	ID      string `path:"id"`
	ShareID string `path:"shareId"`
}

// CreateGameShareEndpoint creates a share link for a game.
//
// A workshop share (workshopId set) plays on the workshop's default API key
// and is reused when the game already has one in that workshop. A personal or
// organisation share needs sponsorKeyShareId; it is an organisation share when
// that key share belongs to an institution.
type CreateGameShareEndpoint struct{}

func (CreateGameShareEndpoint) Method() string             { return "POST" }
func (CreateGameShareEndpoint) Path() string               { return "/api/games/{id}/shares" }
func (CreateGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateGameShareEndpoint) Handle(ctx context.Context, req CreateGameShareReq) (GameShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return GameShareResp{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return GameShareResp{}, err
	}

	game, err := db.GetGameByID(ctx, &user.ID, gameID)
	if err != nil {
		return GameShareResp{}, errNotFound("Game not found")
	}

	var sponsorShareID uuid.UUID
	var institutionID, workshopID *uuid.UUID

	if req.WorkshopID != nil {
		// Workshop share: validate workshop access and use its default key.
		workshop, err := db.GetWorkshopByID(ctx, user.ID, *req.WorkshopID)
		if err != nil {
			return GameShareResp{}, errNotFound("Workshop not found")
		}

		// Head/staff can always share; participants need AllowGameSharing.
		userObj, err := db.GetUserByID(ctx, user.ID)
		if err != nil {
			return GameShareResp{}, fmt.Errorf("failed to load user: %w", err)
		}
		canEditAll := userObj.Role != nil && (userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff || userObj.Role.Role == obj.RoleAdmin)
		if !canEditAll && !workshop.AllowGameSharing {
			return GameShareResp{}, errForbidden("Game sharing is not enabled for this workshop")
		}

		if game.WorkshopID == nil || *game.WorkshopID != *req.WorkshopID {
			return GameShareResp{}, errBadRequest("Game does not belong to this workshop")
		}

		// Non-owners (head/staff) can only share public games; the creator
		// can always share their own game.
		isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
		if !isOwner && !game.Public {
			return GameShareResp{}, errForbidden("Only public games can be shared. The game creator must set it to public first.")
		}

		if existing, err := db.GetWorkshopGameShare(ctx, gameID, *req.WorkshopID); err == nil {
			return toGameShareResp(*existing), nil
		}

		if workshop.DefaultApiKeyShareID == nil {
			return GameShareResp{}, errBadRequest("Workshop has no default API key configured")
		}

		sponsorShareID = *workshop.DefaultApiKeyShareID
		workshopID = req.WorkshopID
		if workshop.Institution != nil {
			institutionID = &workshop.Institution.ID
		}
	} else {
		// Personal or organisation share.
		if req.SponsorKeyShareID == nil {
			return GameShareResp{}, errBadRequest("sponsorKeyShareId is required")
		}
		sponsorShareID = *req.SponsorKeyShareID

		// The owner can share their own game; anyone can share a public game.
		isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == user.ID
		if !isOwner && !game.Public {
			return GameShareResp{}, errForbidden("Only the owner can share a non-public game")
		}

		// A key share belonging to an institution makes this an organisation share.
		apiKeyShare, err := db.GetApiKeyShareByID(ctx, user.ID, sponsorShareID)
		if err != nil {
			return GameShareResp{}, errBadRequest("Invalid API key share")
		}
		if apiKeyShare.Institution != nil {
			institutionID = &apiKeyShare.Institution.ID
		}
	}

	gameShare, err := db.CreateGameShare(ctx, user.ID, gameID, sponsorShareID, institutionID, workshopID, req.MaxSessions, req.AiQualityTier)
	if err != nil {
		log.Warn("failed to create game share", "game_id", gameID, "error", err)
		return GameShareResp{}, appError(err)
	}
	return toGameShareResp(*gameShare), nil
}

// CreateGameShareReq is the game from the path and the share's configuration.
type CreateGameShareReq struct {
	ID string `path:"id"`
	// SponsorKeyShareID is required for personal shares; workshop shares ignore it.
	SponsorKeyShareID *uuid.UUID `json:"sponsorKeyShareId"`
	// MaxSessions is null for unlimited.
	MaxSessions *int `json:"maxSessions"`
	// WorkshopID makes a workshop share, played on the workshop's default key.
	WorkshopID *uuid.UUID `json:"workshopId"`
	// AiQualityTier is null for the source's default (workshop tier or system default).
	AiQualityTier *string `json:"aiQualityTier"`
}

// DeleteGameShareEndpoint revokes a share link and cleans up its guest data.
//
// The share's creator, the game's owner, members of the share's workshop and
// head or staff of its institution may revoke it.
type DeleteGameShareEndpoint struct{}

func (DeleteGameShareEndpoint) Method() string             { return "DELETE" }
func (DeleteGameShareEndpoint) Path() string               { return "/api/games/{id}/shares/{shareId}" }
func (DeleteGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteGameShareEndpoint) Handle(ctx context.Context, req GameShareIDReq) (MessageResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return MessageResp{}, err
	}
	gameID, shareID, err := parseGameShareIDs(req.ID, req.ShareID)
	if err != nil {
		return MessageResp{}, err
	}
	allowed, err := canManageGameShare(ctx, user.ID, gameID, shareID)
	if err != nil {
		return MessageResp{}, err
	}
	if !allowed {
		return MessageResp{}, errForbidden("Not authorized to revoke this share")
	}
	if err := db.DeleteGameShare(ctx, shareID); err != nil {
		log.Warn("failed to delete game share", "game_id", gameID, "share_id", shareID, "error", err)
		return MessageResp{}, fmt.Errorf("failed to delete share: %w", err)
	}
	return MessageResp{Message: "Share deleted"}, nil
}

// UpdateGameShareEndpoint updates a share link's settings.
//
// Whoever may revoke the share may update it.
type UpdateGameShareEndpoint struct{}

func (UpdateGameShareEndpoint) Method() string             { return "PATCH" }
func (UpdateGameShareEndpoint) Path() string               { return "/api/games/{id}/shares/{shareId}" }
func (UpdateGameShareEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateGameShareEndpoint) Handle(ctx context.Context, req UpdateGameShareReq) (GameShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return GameShareResp{}, err
	}
	gameID, shareID, err := parseGameShareIDs(req.ID, req.ShareID)
	if err != nil {
		return GameShareResp{}, err
	}
	allowed, err := canManageGameShare(ctx, user.ID, gameID, shareID)
	if err != nil {
		return GameShareResp{}, err
	}
	if !allowed {
		return GameShareResp{}, errForbidden("Not authorized to update this share")
	}
	updated, err := db.UpdateGameShare(ctx, shareID, req.MaxSessions, req.AiQualityTier)
	if err != nil {
		log.Warn("failed to update game share", "share_id", shareID, "error", err)
		return GameShareResp{}, fmt.Errorf("failed to update share: %w", err)
	}
	return toGameShareResp(*updated), nil
}

// UpdateGameShareReq is the share from the path and its new settings.
type UpdateGameShareReq struct {
	ID      string `path:"id"`
	ShareID string `path:"shareId"`
	// MaxSessions is null for unlimited.
	MaxSessions *int `json:"maxSessions"`
	// AiQualityTier is null for the source's default.
	AiQualityTier *string `json:"aiQualityTier"`
}

// parseGameShareIDs parses the game and share ids from the path.
func parseGameShareIDs(rawGameID, rawShareID string) (uuid.UUID, uuid.UUID, error) {
	gameID, err := parseID(rawGameID, "Invalid game ID")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	shareID, err := parseID(rawShareID, "Invalid share ID")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return gameID, shareID, nil
}

// canManageGameShare says whether the user may update or revoke a share of a
// game: its creator, the game's owner, and for a workshop share head or staff
// of the share's institution and members of its workshop. It refuses a game
// the user cannot see and a share of another game as not found.
func canManageGameShare(ctx context.Context, userID, gameID, shareID uuid.UUID) (bool, error) {
	game, err := db.GetGameByID(ctx, &userID, gameID)
	if err != nil {
		return false, errNotFound("Game not found")
	}
	share, err := db.GetGameShareByID(ctx, shareID)
	if err != nil || share.GameID != gameID {
		return false, errNotFound("Share not found")
	}

	isOwner := game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == userID
	isCreator := share.CreatedBy != nil && *share.CreatedBy == userID
	if isOwner || isCreator {
		return true, nil
	}
	if share.WorkshopID == nil {
		return false, nil
	}
	userObj, err := db.GetUserByID(ctx, userID)
	if err != nil || userObj.Role == nil {
		return false, nil
	}
	if userObj.Role.Institution != nil &&
		share.InstitutionID != nil && *share.InstitutionID == userObj.Role.Institution.ID &&
		(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
		return true, nil
	}
	return userObj.Role.Workshop != nil && *share.WorkshopID == userObj.Role.Workshop.ID, nil
}

// EnrichedGameShare is a game share with its share URL, where it comes from
// and the names it refers to.
type EnrichedGameShare struct {
	obj.GameShare
	ShareURL string `json:"shareUrl"`
	// Source is "workshop", "organization" or "personal".
	Source string `json:"source"`
	// WorkshopName is set for workshop shares.
	WorkshopName string `json:"workshopName,omitempty"`
	// GameName is set where shares of several games are listed together.
	GameName string `json:"gameName,omitempty"`
}

// PrivateShareStatusResp is the shares of a game the caller has access to.
type PrivateShareStatusResp struct {
	Shares []EnrichedGameShare `json:"shares"`
}

// GetPrivateShareStatusEndpoint lists the share links of a game the caller
// has access to.
//
// These are the caller's own personal and organisation shares, their
// institution's shares for head, staff and admins, the shares of the workshop
// they are in, and for head and staff outside a workshop the shares of every
// workshop of their institution.
type GetPrivateShareStatusEndpoint struct{}

func (GetPrivateShareStatusEndpoint) Method() string             { return "GET" }
func (GetPrivateShareStatusEndpoint) Path() string               { return "/api/games/{id}/private-share" }
func (GetPrivateShareStatusEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetPrivateShareStatusEndpoint) Handle(ctx context.Context, req GameIDReq) (PrivateShareStatusResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return PrivateShareStatusResp{}, err
	}
	gameID, err := parseID(req.ID, "Invalid game ID")
	if err != nil {
		return PrivateShareStatusResp{}, err
	}
	if _, err := db.GetGameByID(ctx, &user.ID, gameID); err != nil {
		return PrivateShareStatusResp{}, errNotFound("Game not found")
	}

	userObj, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		return PrivateShareStatusResp{}, fmt.Errorf("failed to load user: %w", err)
	}

	seen := map[uuid.UUID]bool{}
	result := []EnrichedGameShare{}
	add := func(s obj.GameShare, source, workshopName string) {
		if seen[s.ID] {
			return
		}
		seen[s.ID] = true
		result = append(result, EnrichedGameShare{
			GameShare:    s,
			ShareURL:     "/play/" + s.Token,
			Source:       source,
			WorkshopName: workshopName,
		})
	}

	// Personal and organisation shares the caller created outside a workshop.
	personalShares, _ := db.GetGameSharesByGameIDAndCreator(ctx, gameID, user.ID)
	for _, s := range personalShares {
		source := "personal"
		if s.InstitutionID != nil {
			source = "organization"
		}
		add(s, source, "")
	}

	// The institution's shares, for its head, staff and admins.
	if userObj.Role != nil && userObj.Role.Institution != nil &&
		(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff || userObj.Role.Role == obj.RoleAdmin) {
		orgShares, _ := db.GetGameSharesByGameIDAndInstitution(ctx, gameID, userObj.Role.Institution.ID)
		for _, s := range orgShares {
			add(s, "organization", "")
		}
	}

	// The shares of the workshop the caller is in.
	if userObj.Role != nil && userObj.Role.Workshop != nil {
		wsShares, _ := db.GetGameSharesByGameIDAndWorkshop(ctx, gameID, userObj.Role.Workshop.ID)
		for _, s := range wsShares {
			add(s, "workshop", userObj.Role.Workshop.Name)
		}
	}

	// Head and staff outside a workshop see the shares of all their institution's workshops.
	if userObj.Role != nil && userObj.Role.Workshop == nil && userObj.Role.Institution != nil &&
		(userObj.Role.Role == obj.RoleHead || userObj.Role.Role == obj.RoleStaff) {
		workshops, _ := db.ListWorkshopsForInstitution(ctx, userObj.Role.Institution.ID)
		for _, ws := range workshops {
			wsShares, _ := db.GetGameSharesByGameIDAndWorkshop(ctx, gameID, ws.ID)
			for _, s := range wsShares {
				add(s, "workshop", ws.Name)
			}
		}
	}

	return PrivateShareStatusResp{Shares: result}, nil
}
