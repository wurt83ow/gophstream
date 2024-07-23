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

type Log interface {
	Info(string, ...zap.Field)
}

// MemoryStorage represents an in-memory storage with locking mechanisms
type MemoryStorage struct {
	ctx      context.Context
	mx       sync.RWMutex
	messages map[int]models.Message
	keeper   Keeper
	log      Log
}

// Keeper interface for database operations
type Keeper interface {
	InsertMessage(context.Context, models.Message) (int, error)
	GetProcessedMessages(context.Context, models.Filter, models.Pagination) ([]models.Message, error)
	Ping(context.Context) bool
	Close() bool
	UpdateMessageProcessed(context.Context, int) error
}

// NewMemoryStorage creates a new MemoryStorage instance
func NewMemoryStorage(ctx context.Context, keeper Keeper, log Log) *MemoryStorage {
	return &MemoryStorage{
		ctx:      ctx,
		messages: make(map[int]models.Message),
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

// GetProcessedMessages retrieves processed messages from the database based on the provided filter and pagination
func (s *MemoryStorage) GetProcessedMessages(ctx context.Context, filter models.Filter, pagination models.Pagination) ([]models.Message, error) {
	// Get messages from the database
	messages, err := s.keeper.GetProcessedMessages(ctx, filter, pagination)
	if err != nil {
		s.log.Info("error getting processed messages from database: ", zap.Error(err))
		return nil, err
	}

	return messages, nil
}

// UpdateMessageProcessed updates the processed status of a message in the database
func (s *MemoryStorage) UpdateMessageProcessed(ctx context.Context, id int) error {
	// Update the message's processed status in the database
	err := s.keeper.UpdateMessageProcessed(ctx, id)
	if err != nil {
		s.log.Info("error updating message processed status in database: ", zap.Error(err))
		return err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	// Update the in-memory map if the message exists
	if message, exists := s.messages[id]; exists {
		message.Processed = true
		s.messages[id] = message
	} else {
		s.log.Info("message not found in memory", zap.Int("id", id))
		return ErrNotFound
	}

	return nil
}
