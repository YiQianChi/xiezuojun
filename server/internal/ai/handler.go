package ai

// Message is used by ai.Client methods and gateway/memory packages.
type Message struct {
	Role    string `json:"role"`    // user | assistant | system
	Content string `json:"content"`
}
