package db

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Auth0ID string `gorm:"uniqueIndex"` // Unique identifier from Auth0
	Email   string
	Name    string
	// Add other fields as necessary
}

// CreateUser creates a new user in the database
func CreateUser(db *gorm.DB, user *User) error {
	return db.Create(user).Error
}

// GetUserByID gets a user by ID
func GetUserByID(db *gorm.DB, id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}

// GetUserByAuth0ID gets a user by Auth0 ID
func GetUserByAuth0ID(db *gorm.DB, auth0ID string) (*User, error) {
	var user User
	err := db.Where("auth0_id = ?", auth0ID).First(&user).Error
	return &user, err
}

// UpdateUser updates an existing user
func UpdateUser(db *gorm.DB, user *User) error {
	return db.Save(user).Error
}

// DeleteUser deletes a user
func DeleteUser(db *gorm.DB, id uint) error {
	return db.Delete(&User{}, id).Error
}
