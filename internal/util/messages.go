package util

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
