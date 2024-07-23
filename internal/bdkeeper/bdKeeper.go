package bdkeeper

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/wurt83ow/gophstream/internal/models"
	"github.com/wurt83ow/gophstream/internal/storage"
	"go.uber.org/zap"
)

type Log interface {
	Info(string, ...zap.Field)
}

type BDKeeper struct {
	pool *pgxpool.Pool
	log  Log
}

func NewBDKeeper(dsn func() string, log Log, userUpdateInterval func() string) *BDKeeper {
	addr := dsn()
	if addr == "" {
		log.Info("database dsn is empty")
		return nil
	}

	config, err := pgxpool.ParseConfig(addr)
	if err != nil {
		log.Info("Unable to parse database DSN: ", zap.Error(err))
		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Info("Unable to connect to database: ", zap.Error(err))
		return nil
	}

	connConfig, err := pgx.ParseConfig(addr)
	if err != nil {
		log.Info("Unable to parse connection string: %v\n")
	}
	// Register the driver with the name pgx
	sqlDB := stdlib.OpenDB(*connConfig)

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		log.Info("Error getting driver: ", zap.Error(err))
		return nil
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Info("Error getting current directory: ", zap.Error(err))
	}

	// fix error test path
	mp := dir + "/migrations"
	var path string
	if _, err := os.Stat(mp); err != nil {
		path = "../../"
	} else {
		path = dir + "/"
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%smigrations", path),
		"postgres",
		driver)
	if err != nil {
		log.Info("Error creating migration instance: ", zap.Error(err))
		return nil
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Info("Error while performing migration: ", zap.Error(err))
		return nil
	}

	log.Info("Connected!")

	return &BDKeeper{
		pool:               pool,
		log:                log,
		userUpdateInterval: userUpdateInterval,
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

// LoadMessages loads messages from the database
func (kp *BDKeeper) LoadMessages(ctx context.Context) (storage.StorageMessage, error) {
	query := `
    SELECT
        id,
        content,
        created_at,
        processed
    FROM
        messages`

	rows, err := kp.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	data := make(map[int]models.Message)

	for rows.Next() {
		var m models.Message

		err := rows.Scan(
			&m.ID,
			&m.Content,
			&m.CreatedAt,
			&m.Processed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load messages: %w", err)
		}

		data[m.ID] = m
	}

	return data, nil
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

// GetMessages retrieves processed messages from the database based on the provided filter and pagination
func (kp *BDKeeper) GetMessages(ctx context.Context, filter models.Filter, pagination models.Pagination) ([]models.Message, error) {
	var messages []models.Message
	var query string
	var rows pgx.Rows
	var err error

	if filter.Processed == nil {
		query = `SELECT id, content, created_at, processed FROM messages LIMIT $1 OFFSET $2`
		rows, err = kp.pool.Query(ctx, query, pagination.Limit, pagination.Offset)
	} else {
		query = `SELECT id, content, created_at, processed FROM messages WHERE processed = $1 LIMIT $2 OFFSET $3`
		rows, err = kp.pool.Query(ctx, query, *filter.Processed, pagination.Limit, pagination.Offset)
	}

	if err != nil {
		kp.log.Info("Error getting messages from database: ", zap.Error(err))
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

// UpdateMessagesProcessed updates the processed status of messages in the database
func (kp *BDKeeper) UpdateMessagesProcessed(ctx context.Context, ids []int) error {
	query := `UPDATE messages SET processed = true WHERE id = ANY($1)`
	_, err := kp.pool.Exec(ctx, query, ids)
	if err != nil {
		kp.log.Info("Error updating messages processed status in database: ", zap.Error(err))
		return err
	}
	return nil
}
