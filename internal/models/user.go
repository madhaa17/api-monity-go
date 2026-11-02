package models

import "time"

type User struct {
	ID       int64    `gorm:"primaryKey" json:"-"`
	UUID     string   `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	Email    string   `gorm:"uniqueIndex" json:"email"`
	Password string   `json:"-"`
	Name     *string  `json:"name"`
	Role     UserRole `gorm:"type:user_role;default:'USER'" json:"role"`
	// PasswordChangedAt is when the password was last changed; tokens issued earlier are invalid.
	PasswordChangedAt *time.Time `json:"-"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}
