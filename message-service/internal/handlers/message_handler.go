package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"message-service/internal/models"
	"message-service/pkg/db"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type MessageDTO struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

type SendMessageRequest struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
}

type GetMessageResponse struct {
	Messages 		[]MessageDTO					`json:"messages"`
	NextCursor 		*string							`json:"next_cursor,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func encodeCursor(c Cursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return  "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func decodeCursor(d string) (Cursor, error) {
	var c Cursor
	data, err := base64.URLEncoding.DecodeString(d)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(data, &c)
	return c, err
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

func GetMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	conversationID := r.URL.Query().Get("conversation_id")
	cursorStr := r.URL.Query().Get("cursor")

	if conversationID == "" {
		http.Error(w, "conversation_id required", http.StatusBadRequest)
		return 
	}
	
	if _, err := uuid.Parse(conversationID); err != nil {
		http.Error(w, "invalid conversation_id", http.StatusBadRequest)
		return
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = db.DB.QueryContext(
			r.Context(),
			`SELECT id, sender_id, content, created_at
			FROM messages
			where conversation_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT 20`,
			conversationID,
		)
	} else {
		cursor, err := decodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}

		rows, err = db.DB.QueryContext(
			r.Context(),
			`SELECT id, sender_id, content, created_at
			FROM messages
			where conversation_id = $1 
				AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT 20`,
			conversationID,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var messages []MessageDTO
	var lastCursor *Cursor

	for rows.Next() {
		var m MessageDTO

		if err := rows.Scan(&m.ID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return 
		}

		lastCursor = &Cursor{
			CreatedAt: m.CreatedAt,
			ID: m.ID,
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "row iteration error", http.StatusInternalServerError)
		return 
	}

	var nextCursor *string
	if lastCursor != nil {
		encoded, err := encodeCursor(*lastCursor)
		if err != nil {
			http.Error(w, "cursor encoding error", http.StatusInternalServerError)
			return
		}
		nextCursor = &encoded
	}

	resp := GetMessageResponse{
		Messages: messages,
		NextCursor: nextCursor,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}