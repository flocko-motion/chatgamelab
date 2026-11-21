// package: db / database access and repository layer
// type:    data
// job:     create, delete, default-selection, and metadata updates for API keys.
// limits:  does not manage shares (-> api_key_shares.go).
package db

const apiKeyShortenLength = 6
