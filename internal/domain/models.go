package domain

import "time"

type Task struct {
	Key      string
	Title    string
	Epic     string
	Status   string
	Provider string
}

type AuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type,omitempty"`
	ProviderURL  string    `json:"provider_url,omitempty"`
}

func (a AuthToken) ExpiresWithin(d time.Duration) bool {
	if a.ExpiresAt.IsZero() {
		return true
	}
	return time.Until(a.ExpiresAt) <= d
}

type BranchValidationResult struct {
	Valid   bool
	Pattern string
	Branch  string
	Reason  string
}

type CommitMessageContext struct {
	TaskKey   string
	TaskTitle string
	TaskEpic  string
	Summary   string
}
