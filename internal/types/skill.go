package types

// Skill represents a registered skill that can handle events.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}
