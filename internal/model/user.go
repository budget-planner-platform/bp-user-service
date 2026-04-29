package model

import "time"

// User represents a platform user stored in the database.
type User struct {
	ID          string
	Email       string
	DisplayName string
	AvatarURL   string
	Currency    string
	Timezone    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
