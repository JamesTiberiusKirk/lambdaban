package util

import (
	"context"
	"net/http"
)

type MessageType string

const (
	MessageTypeError   MessageType = "error"
	MessageTypeWarning MessageType = "warning"
	MessageTypeInfo    MessageType = "info"
)

// UiMessage holds a typed message string
type UiMessage struct {
	Type    MessageType `json:"type"`
	Content string      `json:"content"`
}

// contextKey is an unexported type for context keys to avoid collisions
type contextKey string

const uiMessagesKey contextKey = "messages"

// AddMessageToRequest adds a message to the request's context and returns the updated request
func AddUiMessageToRequest(r *http.Request, msgType MessageType, msg string) *http.Request {
	// Get existing messages from context
	existingMessages, _ := r.Context().Value(uiMessagesKey).([]UiMessage)

	// Append the new message
	updatedMessages := append(existingMessages, UiMessage{Type: msgType, Content: msg})

	// Create a new context with updated messages slice
	ctx := context.WithValue(r.Context(), uiMessagesKey, updatedMessages)

	// Return a new request with the updated context
	return r.WithContext(ctx)
}

// GetUiMessagesFromRequest retrieves all messages stored in the request context
func GetUiMessagesFromRequest(r *http.Request) []UiMessage {
	messages, ok := r.Context().Value(uiMessagesKey).([]UiMessage)
	if !ok {
		return []UiMessage{}
	}
	return messages
}
