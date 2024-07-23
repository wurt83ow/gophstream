package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/wurt83ow/gophstream/internal/models"
	"go.uber.org/zap"
)

// ErrConflict indicates a data conflict in the store.
var (
	ErrConflict = errors.New("data conflict")
	ErrNotFound = errors.New("not found")
)

type (
	StorageMessage = map[int]models.Message
)

type Log interface {
	Info(string, ...zap.Field)
}

// MemoryStorage represents an in-memory storage with locking mechanisms
type MemoryStorage struct {
	ctx      context.Context
	mx       sync.RWMutex
	messages StorageMessage
	keeper   Keeper
	log      Log
}

// Keeper interface for database operations
type Keeper interface {
	LoadMessages(context.Context) (StorageMessage, error)
	InsertMessage(context.Context, models.Message) (int, error)
	GetMessages(context.Context, models.Filter, models.Pagination) ([]models.Message, error)
	UpdateMessagesProcessed(ctx context.Context, ids []int) error
	Ping(context.Context) bool
	Close() bool
}

// NewMemoryStorage creates a new MemoryStorage instance
func NewMemoryStorage(ctx context.Context, keeper Keeper, log Log) *MemoryStorage {
	messages := make(StorageMessage)

	if keeper != nil {
		var err error
		// Load messages
		messages, err = keeper.LoadMessages(ctx)
		if err != nil {
			log.Info("cannot load user data: ", zap.Error(err))
		}
	}

	return &MemoryStorage{
		ctx:      ctx,
		messages: messages,
		keeper:   keeper,
		log:      log,
	}
}

// InsertMessage inserts a new message into the storage and database
func (s *MemoryStorage) InsertMessage(ctx context.Context, message models.Message) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	// Insert message into the database
	id, err := s.keeper.InsertMessage(ctx, message)
	if err != nil {
		s.log.Info("error inserting message to database: ", zap.Error(err))
		return err
	}

	// Save to the in-memory map with the new ID
	message.ID = id
	s.messages[id] = message

	return nil
}

// GetMessages retrieves processed messages from the database based on the provided filter and pagination
func (s *MemoryStorage) GetMessages(ctx context.Context, filter models.Filter, pagination models.Pagination) ([]models.Message, error) {
	// Get messages from the database
	messages, err := s.keeper.GetMessages(ctx, filter, pagination)
	if err != nil {
		s.log.Info("error getting processed messages from database: ", zap.Error(err))
		return nil, err
	}

	return messages, nil
}

// UpdateMessagesProcessed updates the processed status of messages in the database
func (s *MemoryStorage) UpdateMessagesProcessed(ctx context.Context, ids []int) error {
	// Update the messages' processed status in the database
	err := s.keeper.UpdateMessagesProcessed(ctx, ids)
	if err != nil {
		s.log.Info("error updating messages processed status in database: ", zap.Error(err))
		return err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	// Update the in-memory map if the messages exist
	for _, id := range ids {
		if message, exists := s.messages[id]; exists {
			message.Processed = true
			s.messages[id] = message
		} else {
			s.log.Info("message not found in memory", zap.Int("id", id))
			return ErrNotFound
		}
	}

	return nil
}
