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
	MessageTypeStream    MessageType = "stream_event"
)

// Message is the interface that all message types implement
type Message interface {
	Type() MessageType
}

// ContentBlock represents different types of content in a message
type ContentBlock struct {
	Type     string       `json:"type"`
	Text     *string      `json:"text,omitempty"`
	Tool     *ToolUse     `json:"-"`
	Result   *ToolResult  `json:"-"`
	Thinking *ThinkingBlock `json:"-"`
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

// ThinkingBlock represents a thinking content block (extended thinking)
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (c ContentBlock) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "text":
		text := ""
		if c.Text != nil {
			text = *c.Text
		}
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{Type: c.Type, Text: text})

	case "tool_use":
		if c.Tool == nil {
			return nil, fmt.Errorf("tool_use content block missing Tool data")
		}
		return json.Marshal(struct {
			Type  string         `json:"type"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		}{Type: c.Type, ID: c.Tool.ID, Name: c.Tool.Name, Input: c.Tool.Input})

	case "tool_result":
		if c.Result == nil {
			return nil, fmt.Errorf("tool_result content block missing Result data")
		}
		return json.Marshal(struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   any    `json:"content,omitempty"`
			IsError   *bool  `json:"is_error,omitempty"`
		}{Type: c.Type, ToolUseID: c.Result.ToolUseID, Content: c.Result.Content, IsError: c.Result.IsError})

	case "thinking":
		if c.Thinking == nil {
			return nil, fmt.Errorf("thinking content block missing Thinking data")
		}
		return json.Marshal(struct {
			Type      string `json:"type"`
			Thinking  string `json:"thinking"`
			Signature string `json:"signature"`
		}{Type: c.Type, Thinking: c.Thinking.Thinking, Signature: c.Thinking.Signature})

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
		Thinking  string         `json:"thinking,omitempty"`
		Signature string         `json:"signature,omitempty"`
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
	case "thinking":
		c.Thinking = &ThinkingBlock{
			Thinking:  raw.Thinking,
			Signature: raw.Signature,
		}
	}

	return nil
}

// UserMessage represents a message from the user
type UserMessage struct {
	Content         string `json:"content"`
	UUID            string `json:"uuid,omitempty"`
	ParentToolUseID string `json:"parent_tool_use_id,omitempty"`
}

func (m *UserMessage) Type() MessageType { return MessageTypeUser }

// AssistantMessage represents a message from Claude
type AssistantMessage struct {
	Content         []ContentBlock        `json:"content"`
	Model           string                `json:"model,omitempty"`
	ParentToolUseID string                `json:"parent_tool_use_id,omitempty"`
	Error           AssistantMessageError `json:"error,omitempty"`
}

func (m *AssistantMessage) Type() MessageType { return MessageTypeAssistant }

// SystemMessage represents a system message
type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"`
}

func (m *SystemMessage) Type() MessageType { return MessageTypeSystem }

// ResultMessage represents the final result of a conversation
type ResultMessage struct {
	Subtype          string         `json:"subtype"`
	DurationMS       int            `json:"duration_ms"`
	DurationAPIMS    int            `json:"duration_api_ms"`
	IsError          bool           `json:"is_error"`
	NumTurns         int            `json:"num_turns"`
	SessionID        string         `json:"session_id"`
	TotalCostUSD     *float64       `json:"total_cost_usd,omitempty"`
	Usage            map[string]any `json:"usage,omitempty"`
	Result           *string        `json:"result,omitempty"`
	StructuredOutput any            `json:"structured_output,omitempty"`
}

func (m *ResultMessage) Type() MessageType { return MessageTypeResult }

// StreamEvent represents a partial message update during streaming
type StreamEvent struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Event           map[string]any `json:"event"` // Raw Anthropic API stream event
	ParentToolUseID string         `json:"parent_tool_use_id,omitempty"`
}

func (m *StreamEvent) Type() MessageType { return MessageTypeStream }

// ParseMessage parses a raw message from the CLI into a typed Message
func ParseMessage(data map[string]any) (Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing or invalid type field", ErrInvalidMessage)
	}

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
		return &msg, nil

	case MessageTypeAssistant:
		msgData, ok := data["message"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: invalid assistant message structure", ErrInvalidMessage)
		}
		content, ok := msgData["content"].([]any)
		if !ok {
			return nil, fmt.Errorf("%w: invalid assistant message content", ErrInvalidMessage)
		}
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

		assistantMsg := &AssistantMessage{Content: blocks}

		// Extract model if present
		if model, ok := msgData["model"].(string); ok {
			assistantMsg.Model = model
		}

		// Extract parent_tool_use_id if present
		if parentID, ok := data["parent_tool_use_id"].(string); ok {
			assistantMsg.ParentToolUseID = parentID
		}

		// Extract error if present
		if errField, ok := data["error"].(string); ok {
			assistantMsg.Error = AssistantMessageError(errField)
		}

		return assistantMsg, nil

	case MessageTypeSystem:
		var msg SystemMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse system message: %v", ErrInvalidMessage, err)
		}
		return &msg, nil

	case MessageTypeResult:
		var msg ResultMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse result message: %v", ErrInvalidMessage, err)
		}
		return &msg, nil

	case MessageTypeStream:
		var msg StreamEvent
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			return nil, fmt.Errorf("%w: failed to parse stream event: %v", ErrInvalidMessage, err)
		}
		return &msg, nil

	default:
		return nil, fmt.Errorf("%w: unknown message type: %s", ErrInvalidMessage, msgType)
	}
}
