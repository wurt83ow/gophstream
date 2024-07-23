package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Log interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, err error, keysAndValues ...interface{})
}

type KafkaProducerImpl struct {
	writer *kafka.Writer
	log    Log
}

func NewKafkaProducer(ctx context.Context, brokers []string, topic string, log Log) *KafkaProducerImpl {
	kp := &KafkaProducerImpl{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
		log: log,
	}

	kp.log.Info("Kafka producer created", "topic", topic, "brokers", brokers)
	return kp
}

func (kp *KafkaProducerImpl) SendMessage(ctx context.Context, topic string, message []byte) error {
	kp.log.Info("Sending message", "topic", topic, "message", string(message))

	err := kp.writer.WriteMessages(ctx,
		kafka.Message{
			Topic: topic,
			Value: message,
		},
	)

	if err != nil {
		kp.log.Error("Failed to send message", err, "topic", topic, "message", string(message))
		return err
	}

	kp.log.Info("Message sent successfully", "topic", topic, "message", string(message))
	return nil
}
