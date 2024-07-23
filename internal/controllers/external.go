package controllers

import (
	"context"
	"encoding/json"

	"github.com/wurt83ow/gophstream/internal/models"
	"go.uber.org/zap"
)

// KafkaProducer interface for Kafka operations
type KafkaProducer interface {
	SendMessage(topic string, message []byte) error
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

// SendMessageToKafka sends a message to Kafka and returns the ID of the sent message
func (c *ExtController) SendMessageToKafka(message models.Message) (int, error) {
	// Marshal the message to JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		c.log.Info("error marshaling message: ", zap.Error(err))
		return 0, err
	}

	// Send the message to Kafka
	if err := c.kafka.SendMessage("message_topic", messageData); err != nil {
		c.log.Info("error sending message to Kafka: ", zap.Error(err))
		return 0, err
	}

	c.log.Info("Message sent to Kafka successfully", zap.Int("messageID", message.ID))
	return message.ID, nil
}
