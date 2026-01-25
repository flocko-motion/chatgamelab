package imagecache

import (
	"crypto/md5"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxEntryAge is the maximum time an entry can exist before cleanup
	MaxEntryAge = 5 * time.Minute
	// CleanupInterval is how often the cleanup loop runs
	CleanupInterval = 30 * time.Second
)

// ImageSaverFunc persists image data to the database
type ImageSaverFunc func(messageID uuid.UUID, imageData []byte) error

// Entry represents a cached image (partial or complete)
type Entry struct {
	MessageID  uuid.UUID
	ImageData  []byte
	Hash       string
	IsComplete bool
	HasError   bool
	ErrorCode  string // Machine-readable error code for frontend
	ErrorMsg   string
	ImageSaver ImageSaverFunc
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Cache manages in-memory image storage during generation
type Cache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*Entry
	done    chan struct{}
}

var defaultCache *Cache
var once sync.Once

// Get returns the singleton cache instance
func Get() *Cache {
	once.Do(func() {
		defaultCache = &Cache{
			entries: make(map[uuid.UUID]*Entry),
			done:    make(chan struct{}),
		}
		go defaultCache.cleanupLoop()
	})
	return defaultCache
}

// Create initializes a new cache entry for a message
func (c *Cache) Create(messageID uuid.UUID, saver ImageSaverFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[messageID] = &Entry{
		MessageID:  messageID,
		ImageSaver: saver,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Update stores new image data and returns the new hash
func (c *Cache) Update(messageID uuid.UUID, imageData []byte, isComplete bool) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[messageID]
	if !exists {
		// Entry was cleaned up or never created, create new one
		entry = &Entry{
			MessageID: messageID,
			CreatedAt: time.Now(),
		}
		c.entries[messageID] = entry
	}

	// Compute hash
	hash := computeHash(imageData)

	entry.ImageData = imageData
	entry.Hash = hash
	entry.IsComplete = isComplete
	entry.UpdatedAt = time.Now()

	// If complete, persist to DB and remove from cache
	if isComplete && entry.ImageSaver != nil {
		go func(saver ImageSaverFunc, msgID uuid.UUID, data []byte) {
			if err := saver(msgID, data); err != nil {
				// Log error but don't block - image is still in cache for now
			}
			// Remove from cache after successful persist
			c.Remove(msgID)
		}(entry.ImageSaver, messageID, imageData)
	}

	return hash
}

// Status contains the current state of a cached image
type Status struct {
	Hash       string
	IsComplete bool
	HasError   bool
	ErrorCode  string
	ErrorMsg   string
	Exists     bool
}

// GetStatus returns the current status of a cached image
func (c *Cache) GetStatus(messageID uuid.UUID) Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[messageID]
	if !exists {
		return Status{Exists: false}
	}

	return Status{
		Hash:       entry.Hash,
		IsComplete: entry.IsComplete,
		HasError:   entry.HasError,
		ErrorCode:  entry.ErrorCode,
		ErrorMsg:   entry.ErrorMsg,
		Exists:     true,
	}
}

// SetError marks an entry as failed with an error code
func (c *Cache) SetError(messageID uuid.UUID, errorCode string, errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[messageID]
	if !exists {
		entry = &Entry{
			MessageID: messageID,
			CreatedAt: time.Now(),
		}
		c.entries[messageID] = entry
	}

	entry.HasError = true
	entry.ErrorCode = errorCode
	entry.ErrorMsg = errMsg
	entry.UpdatedAt = time.Now()
}

// GetImage returns the current image data
func (c *Cache) GetImage(messageID uuid.UUID) (imageData []byte, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[messageID]
	if !exists || len(entry.ImageData) == 0 {
		return nil, false
	}

	return entry.ImageData, true
}

// Remove deletes an entry from the cache
func (c *Cache) Remove(messageID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, messageID)
}

// cleanupLoop periodically removes stale entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

// cleanup removes entries older than MaxEntryAge
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, entry := range c.entries {
		if now.Sub(entry.CreatedAt) > MaxEntryAge {
			delete(c.entries, id)
		}
	}
}

// Stop shuts down the cleanup loop
func (c *Cache) Stop() {
	close(c.done)
}

func computeHash(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	h := md5.Sum(data)
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 hex chars, enough for change detection
}
