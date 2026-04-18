package handlers

import (
	"encoding/json"
	"log"
	"message-service/internal/models"
	"message-service/pkg/db"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type SendMessageRequest struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req SendMessageRequest

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request payload"})
		return
	}

	if req.ConversationID == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "ConversationID and Content are required"})
		return
	}

	if _, err := uuid.Parse(req.ConversationID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid conversation_id"})
		return
	}

	senderID := r.Header.Get("x-user-id")
	if senderID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
		return
	}

	messageID := uuid.New().String()
	var createdAt time.Time

	err := db.DB.QueryRowContext(
		r.Context(),
		`INSERT INTO messages (id, conversation_id, sender_id, content)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		messageID,
		req.ConversationID,
		senderID,
		req.Content,
	).Scan(&createdAt)

	if err != nil {
		log.Printf("DB insert error: %v | conversation_id=%s sender_id=%s",
			err, req.ConversationID, senderID)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	message := models.Message{
		ID:             messageID,
		ConversationID: req.ConversationID,
		SenderID:       senderID,
		Content:        req.Content,
		CreatedAt:      createdAt,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}