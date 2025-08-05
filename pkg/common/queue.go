package common

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// QueueItem represents a single item in the music queue
type QueueItem struct {
	URL         string
	Title       string
	RequestedBy string
	AddedAt     time.Time
}

// MusicQueue manages the queue for a specific guild
type MusicQueue struct {
	guildID   string
	items     []*QueueItem
	current   *QueueItem
	isPlaying bool
	mu        sync.RWMutex
	voiceConn *discordgo.VoiceConnection
	pipeline  *AudioPipeline
}

// NewMusicQueue creates a new music queue for a guild
func NewMusicQueue(guildID string) *MusicQueue {
	return &MusicQueue{
		guildID: guildID,
		items:   make([]*QueueItem, 0),
	}
}

// Add adds a new item to the queue
func (mq *MusicQueue) Add(url, title, requestedBy string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	item := &QueueItem{
		URL:         url,
		Title:       title,
		RequestedBy: requestedBy,
		AddedAt:     time.Now(),
	}

	mq.items = append(mq.items, item)
	log.Printf("Added '%s' to queue for guild %s", title, mq.guildID)
}

// Next gets the next item from the queue
func (mq *MusicQueue) Next() *QueueItem {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.items) == 0 {
		return nil
	}

	item := mq.items[0]
	mq.items = mq.items[1:]
	mq.current = item
	return item
}

// Current returns the currently playing item
func (mq *MusicQueue) Current() *QueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.current
}

// List returns all items in the queue
func (mq *MusicQueue) List() []*QueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	result := make([]*QueueItem, len(mq.items))
	copy(result, mq.items)
	return result
}

// Size returns the number of items in the queue
func (mq *MusicQueue) Size() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return len(mq.items)
}

// Clear clears the entire queue
func (mq *MusicQueue) Clear() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.items = make([]*QueueItem, 0)
	mq.current = nil
}

// Remove removes an item at the specified index
func (mq *MusicQueue) Remove(index int) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if index < 0 || index >= len(mq.items) {
		return fmt.Errorf("invalid index: %d", index)
	}

	removed := mq.items[index]
	mq.items = append(mq.items[:index], mq.items[index+1:]...)
	log.Printf("Removed '%s' from queue for guild %s", removed.Title, mq.guildID)
	return nil
}

// SetPlaying sets the playing state
func (mq *MusicQueue) SetPlaying(playing bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.isPlaying = playing
}

// IsPlaying returns whether something is currently playing
func (mq *MusicQueue) IsPlaying() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.isPlaying
}

// SetVoiceConnection sets the voice connection for this queue
func (mq *MusicQueue) SetVoiceConnection(vc *discordgo.VoiceConnection) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.voiceConn = vc
}

// GetVoiceConnection returns the voice connection
func (mq *MusicQueue) GetVoiceConnection() *discordgo.VoiceConnection {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.voiceConn
}

// SetPipeline sets the audio pipeline for this queue
func (mq *MusicQueue) SetPipeline(pipeline *AudioPipeline) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.pipeline = pipeline
}

// GetPipeline returns the audio pipeline
func (mq *MusicQueue) GetPipeline() *AudioPipeline {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.pipeline
}
