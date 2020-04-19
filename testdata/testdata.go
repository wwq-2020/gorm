package testdata

import "time"

// User user
type User struct {
	ID        int64     `gorm:"id"`
	Name      string    `gorm:"name"`
	Password  string    `gorm:"password"`
	CreatedAt time.Time `gorm:"createdAt"`
}
