package claudecode

import (
	"encoding/json"
	"fmt"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeUser      MessageType = "user"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeSystem    MessageType = "system"
	MessageTypeResult    MessageType = "result"
)

// Message is the interface that all message types implement
type Message interface {
	Type() MessageType
}

// BaseMessage contains common fields for messages
type BaseMessage struct {
	MessageType MessageType `json:"type"`
	SessionID   string      `json:"session_id,omitempty"`
}

// Type returns the message type
func (m BaseMessage) Type() MessageType {
	return m.MessageType
}

// ContentBlock represents different types of content in a message
type ContentBlock struct {
	Type   string      `json:"type"`
	Text   *string     `json:"text,omitempty"`
	Tool   *ToolUse    `json:"-"`
	Result *ToolResult `json:"-"`
}

// ToolUse represents a tool invocation
type ToolUse struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content,omitempty"`
	IsError   *bool  `json:"is_error,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (c ContentBlock) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "text":
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			Type: c.Type,
			Text: *c.Text,
		})
	case "tool_use":
		return json.Marshal(struct {
			Type  string         `json:"type"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		}{
			Type:  c.Type,
			ID:    c.Tool.ID,
			Name:  c.Tool.Name,
			Input: c.Tool.Input,
		})
	case "tool_result":
		return json.Marshal(struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   any    `json:"content,omitempty"`
			IsError   *bool  `json:"is_error,omitempty"`
		}{
			Type:      c.Type,
			ToolUseID: c.Result.ToolUseID,
			Content:   c.Result.Content,
			IsError:   c.Result.IsError,
		})
	default:
		return nil, fmt.Errorf("unknown content block type: %s", c.Type)
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for ContentBlock
func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type      string         `json:"type"`
		Text      *string        `json:"text,omitempty"`
		ID        string         `json:"id,omitempty"`
		Name      string         `json:"name,omitempty"`
		Input     map[string]any `json:"input,omitempty"`
		ToolUseID string         `json:"tool_use_id,omitempty"`
		Content   any            `json:"content,omitempty"`
		IsError   *bool          `json:"is_error,omitempty"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Type = raw.Type

	switch raw.Type {
	case "text":
		c.Text = raw.Text
	case "tool_use":
		c.Tool = &ToolUse{
			ID:    raw.ID,
			Name:  raw.Name,
			Input: raw.Input,
		}
	case "tool_result":
		c.Result = &ToolResult{
			ToolUseID: raw.ToolUseID,
			Content:   raw.Content,
			IsError:   raw.IsError,
		}
	}

	return nil
}

// UserMessage represents a message from the user
type UserMessage struct {
	BaseMessage
	Content string `json:"content"`
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) *UserMessage {
	return &UserMessage{
		BaseMessage: BaseMessage{MessageType: MessageTypeUser},
		Content:     content,
	}
}

// AssistantMessage represents a message from Claude
type AssistantMessage struct {
	BaseMessage
	Content []ContentBlock `json:"content"`
}

// SystemMessage represents a system message
type SystemMessage struct {
	BaseMessage
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"`
}

// ResultMessage represents the final result of a conversation
type ResultMessage struct {
	BaseMessage
	Subtype       string         `json:"subtype"`
	DurationMS    int            `json:"duration_ms"`
	DurationAPIMS int            `json:"duration_api_ms"`
	IsError       bool           `json:"is_error"`
	NumTurns      int            `json:"num_turns"`
	SessionID     string         `json:"session_id"`
	TotalCostUSD  *float64       `json:"total_cost_usd,omitempty"`
	Usage         map[string]any `json:"usage,omitempty"`
	Result        *string        `json:"result,omitempty"`
}

// MessageResult wraps a message with a potential error
type MessageResult struct {
	Message Message
	Error   error
}

// ParseMessage parses a raw message from the CLI into a typed Message
func ParseMessage(data map[string]any) (Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing or invalid type field", ErrInvalidMessage)
	}

	// Marshal back to JSON for proper unmarshaling
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to re-marshal: %v", ErrInvalidMessage, err)
	}

	switch MessageType(msgType) {
	case MessageTypeUser:
		var msg UserMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse user message: %v", ErrInvalidMessage, err)
		}
		msg.MessageType = MessageTypeUser
		return &msg, nil

	case MessageTypeAssistant:
		// Handle the nested message structure from CLI
		if msgData, ok := data["message"].(map[string]any); ok {
			if content, ok := msgData["content"].([]any); ok {
				var blocks []ContentBlock
				for _, item := range content {
					blockJSON, err := json.Marshal(item)
					if err != nil {
						continue
					}
					var block ContentBlock
					if err := json.Unmarshal(blockJSON, &block); err != nil {
						continue
					}
					blocks = append(blocks, block)
				}
				return &AssistantMessage{
					BaseMessage: BaseMessage{MessageType: MessageTypeAssistant},
					Content:     blocks,
				}, nil
			}
		}
		return nil, fmt.Errorf("%w: invalid assistant message structure", ErrInvalidMessage)

	case MessageTypeSystem:
		var msg SystemMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse system message: %v", ErrInvalidMessage, err)
		}
		msg.MessageType = MessageTypeSystem
		return &msg, nil

	case MessageTypeResult:
		var msg ResultMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse result message: %v", ErrInvalidMessage, err)
		}
		msg.MessageType = MessageTypeResult
		return &msg, nil

	default:
		return nil, fmt.Errorf("%w: unknown message type: %s", ErrInvalidMessage, msgType)
	}
}
