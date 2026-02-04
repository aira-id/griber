package domain

// ConversationState tracks conversation history and state
type ConversationState struct {
	ID    string
	Items map[string]*Item // itemID -> Item
	Order []string         // ordered item IDs
}

// Item represents a conversation item
type Item struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`         // "realtime.item"
	Type      string        `json:"type"`           // "message", "function_call"
	Status    string        `json:"status"`         // "in_progress", "completed"
	Role      string        `json:"role,omitempty"` // "user", "assistant"
	Content   []ContentPart `json:"content"`
	CreatedAt int64         `json:"created_at,omitempty"`
}

// ContentPart represents content within an item
type ContentPart struct {
	Type         string        `json:"type"` // "input_text", "input_audio", "text", "output_audio", "function_call"
	Text         string        `json:"text,omitempty"`
	Audio        string        `json:"audio,omitempty"` // base64-encoded audio
	Transcript   string        `json:"transcript,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	Format       string        `json:"format,omitempty"` // "pcm16" for audio
}

// FunctionCall represents a function call in content
type FunctionCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// NewConversationState creates a new conversation state
func NewConversationState(conversationID string) *ConversationState {
	return &ConversationState{
		ID:    conversationID,
		Items: make(map[string]*Item),
		Order: []string{},
	}
}

// AddItem adds an item to the conversation
func (cs *ConversationState) AddItem(item *Item) {
	cs.Items[item.ID] = item
	cs.Order = append(cs.Order, item.ID)
}

// GetItem retrieves an item by ID
func (cs *ConversationState) GetItem(itemID string) *Item {
	return cs.Items[itemID]
}

// DeleteItem removes an item from the conversation
func (cs *ConversationState) DeleteItem(itemID string) bool {
	if _, exists := cs.Items[itemID]; !exists {
		return false
	}
	delete(cs.Items, itemID)
	for i, id := range cs.Order {
		if id == itemID {
			cs.Order = append(cs.Order[:i], cs.Order[i+1:]...)
			break
		}
	}
	return true
}

// NewItem creates a new conversation item
func NewItem(itemID, itemType, role string) *Item {
	return &Item{
		ID:        itemID,
		Object:    "realtime.item",
		Type:      itemType,
		Status:    "in_progress",
		Role:      role,
		Content:   []ContentPart{},
		CreatedAt: 0, // Will be set when needed
	}
}
