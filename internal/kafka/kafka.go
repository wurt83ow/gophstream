package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type KafkaProducerImpl struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducerImpl {
	return &KafkaProducerImpl{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (kp *KafkaProducerImpl) SendMessage(topic string, message []byte) error {
	return kp.writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: message,
		},
	)
}
