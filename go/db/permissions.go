// package: db / database access and repository layer
// type:    logic
// job:     evaluate whether a user may perform CRUD on institutions, workshops, users, and invites by role.
// limits:  does not authorize games/keys (-> permissions_game.go) or define SQL (-> db/queries).
package db

// CRUDOperation represents the type of operation being performed
type CRUDOperation string

const (
	OpCreate CRUDOperation = "create"
	OpRead   CRUDOperation = "read"
	OpUpdate CRUDOperation = "update"
	OpDelete CRUDOperation = "delete"
	OpList   CRUDOperation = "list"
)
