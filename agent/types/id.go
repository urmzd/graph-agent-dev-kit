package types

import "github.com/google/uuid"

// NewID generates a new unique ID.
func NewID() string {
	return uuid.New().String()
}
