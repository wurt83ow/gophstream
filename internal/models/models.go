package models

import "time"

// RequestMessage represents the incoming message data from the client
type RequestMessage struct {
	Content string `json:"content"`
}

// Message represents the message stored in the database and processed through Kafka
type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Processed bool      `json:"processed"`
}

// Filter represents the criteria for filtering messages
type Filter struct {
	Processed bool `json:"processed"`
}

// Pagination represents pagination details for listing messages
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
