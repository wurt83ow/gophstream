package bdkeeper

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wurt83ow/gophstream/internal/models"
	"go.uber.org/zap"
)

type Log interface {
	Info(string, ...zap.Field)
}

type BDKeeper struct {
	pool *pgxpool.Pool
	log  Log
}

func NewBDKeeper(dsn string, log Log) *BDKeeper {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Info("Unable to parse database DSN: ", zap.Error(err))
		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Info("Unable to connect to database: ", zap.Error(err))
		return nil
	}

	log.Info("Connected to database")

	return &BDKeeper{
		pool: pool,
		log:  log,
	}
}

func (kp *BDKeeper) Close() bool {
	if kp.pool != nil {
		kp.pool.Close()
		kp.log.Info("Database connection pool closed")
		return true
	}
	kp.log.Info("Attempted to close a nil database connection pool")
	return false
}

func (kp *BDKeeper) Ping(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	if err := kp.pool.Ping(ctx); err != nil {
		return false
	}

	return true
}

// InsertMessage inserts a new message into the database
func (kp *BDKeeper) InsertMessage(ctx context.Context, message models.Message) (int, error) {
	var id int
	query := `INSERT INTO messages (content, created_at, processed) VALUES ($1, $2, $3) RETURNING id`
	err := kp.pool.QueryRow(ctx, query, message.Content, message.CreatedAt, message.Processed).Scan(&id)
	if err != nil {
		kp.log.Info("Error inserting message to database: ", zap.Error(err))
		return 0, err
	}
	return id, nil
}

// GetProcessedMessages retrieves processed messages from the database based on the provided filter and pagination
func (kp *BDKeeper) GetProcessedMessages(ctx context.Context, filter models.Filter, pagination models.Pagination) ([]models.Message, error) {
	var messages []models.Message
	query := `SELECT id, content, created_at, processed FROM messages WHERE processed = $1 LIMIT $2 OFFSET $3`
	rows, err := kp.pool.Query(ctx, query, filter.Processed, pagination.Limit, pagination.Offset)
	if err != nil {
		kp.log.Info("Error getting processed messages from database: ", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message models.Message
		err := rows.Scan(&message.ID, &message.Content, &message.CreatedAt, &message.Processed)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return messages, nil
}
