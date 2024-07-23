package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wurt83ow/gophstream/internal/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Storage interface for database operations
type Storage interface {
	InsertMessage(context.Context, models.Message) error
	GetProcessedMessages(context.Context, models.Filter, models.Pagination) ([]models.Message, error)
}

// Log interface for logging
type Log interface {
	Info(string, ...zapcore.Field)
}

// BaseController struct for handling requests
type BaseController struct {
	ctx     context.Context
	storage Storage
	log     Log
}

// NewBaseController creates a new BaseController instance
func NewBaseController(ctx context.Context, storage Storage, log Log) *BaseController {
	return &BaseController{
		ctx:     ctx,
		storage: storage,
		log:     log,
	}
}

// Route sets up the routes for the BaseController
func (h *BaseController) Route() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/message", h.AddMessage)
	r.Get("/api/messages", h.GetProcessedMessages)
	return r
}

// @Summary Add message
// @Description Add a new message to the database
// @Tags Messages
// @Accept json
// @Produce json
// @Param message body models.RequestMessage true "Message Info"
// @Success 200 {string} string "Message added to the database successfully"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/message [post]
func (h *BaseController) AddMessage(w http.ResponseWriter, r *http.Request) {
	var msg models.RequestMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		h.log.Info("cannot decode request JSON body: ", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	message := models.Message{
		Content:   msg.Content,
		CreatedAt: time.Now(),
		Processed: false,
	}

	if err := h.storage.InsertMessage(h.ctx, message); err != nil {
		h.log.Info("error inserting message to storage: ", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Message added to the database successfully")); err != nil {
		h.log.Info("error writing response: ", zap.Error(err))
	}
	h.log.Info("Message added to the database successfully")
}

// @Summary Get processed messages
// @Description Get processed messages from the database
// @Tags Messages
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} models.Message "List of processed messages"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/messages [get]
func (h *BaseController) GetProcessedMessages(w http.ResponseWriter, r *http.Request) {
	var filter models.Filter
	var pagination models.Pagination

	if v := r.URL.Query().Get("limit"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			h.log.Info("invalid limit format")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pagination.Limit = val
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			h.log.Info("invalid offset format")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pagination.Offset = val
	}

	messages, err := h.storage.GetProcessedMessages(h.ctx, filter, pagination)
	if err != nil {
		h.log.Info("error getting processed messages from storage: ", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		h.log.Info("error encoding response: ", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}
