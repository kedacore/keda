package notifications

// EntityScopeTypeInput - Scope type for destinations
type EntityScopeTypeInput string

var EntityScopeTypeInputTypes = struct {
	ORGANIZATION EntityScopeTypeInput
	ACCOUNT      EntityScopeTypeInput
}{
	ORGANIZATION: "ORGANIZATION",
	ACCOUNT:      "ACCOUNT",
}

// EntityScopeInput - Scope input for destinations
type EntityScopeInput struct {
	ID   string               `json:"id"`
	Type EntityScopeTypeInput `json:"type"`
}

// EntityScope - Scope response from API
type EntityScope struct {
	ID   string               `json:"id,omitempty"`
	Type EntityScopeTypeInput `json:"type,omitempty"`
}

// feat(notifications): Support for organization and account scoped destinations
