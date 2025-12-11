package db

import (
	"context"
	"log"

	"github.com/google/uuid"
)

// DevUserID is the well-known UUID for the dev user
var DevUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// Run ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	// Ensure dev user exists
	_, err := GetUserByID(ctx, DevUserID)
	if err != nil {
		log.Printf("Creating dev user with ID %s", DevUserID)
		_, err = CreateUserWithID(ctx, DevUserID, "dev", nil, "")
		if err != nil {
			log.Printf("Warning: failed to create dev user: %v", err)
		}
	}
}
