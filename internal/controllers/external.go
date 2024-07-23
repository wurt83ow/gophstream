package controllers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wurt83ow/gophstream/internal/models"
	"go.uber.org/zap"
)

// KafkaProducer interface for Kafka operations
type KafkaProducer interface {
	SendMessage(topic string, message []byte) error
}

// Log interface for logging
type Log interface {
	Info(string, ...zap.Field)
}

type ExtController struct {
	ctx     context.Context
	storage Storage
	kafka   KafkaProducer
	log     Log
}

func NewExtController(ctx context.Context, storage Storage, kafka KafkaProducer, log Log) *ExtController {
	return &ExtController{
		ctx:     ctx,
		storage: storage,
		kafka:   kafka,
		log:     log,
	}
}

func (c *ExtController) SendMessageToKafka(content string) error {
	message := models.Message{
		Content:   content,
		CreatedAt: time.Now(),
		Processed: false,
	}

	// Marshal the message to JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		c.log.Info("error marshaling message: ", zap.Error(err))
		return err
	}

	// Send the message to Kafka
	if err := c.kafka.SendMessage("message_topic", messageData); err != nil {
		c.log.Info("error sending message to Kafka: ", zap.Error(err))
		return err
	}

	c.log.Info("Message sent to Kafka successfully")
	return nil
}
