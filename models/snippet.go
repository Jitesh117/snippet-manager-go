package models

import (
	"time"

	"github.com/google/uuid"
)

type Snippet struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Language    string     `json:"language"`
	Code        string     `json:"code"`
	UserID      uuid.UUID  `json:"user_id"`
	FolderID    *uuid.UUID `json:"folder_id"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Folder struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id"` // Pointer as null values will also be used for root folders
	UserID    uuid.UUID  `json:"user_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // cuz it's a password, duh
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
