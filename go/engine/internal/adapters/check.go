package adapters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrKeyRejected is a provider refusing the key, as opposed to failing to answer.
var ErrKeyRejected = errors.New("the provider rejected the key")

// ErrUnreachable is a provider that could not be asked, or failed to answer.
var ErrUnreachable = errors.New("the provider is unreachable")

// ErrTransient is a failure that may pass if the call is made again: a
// timeout, a rate limit, a server error, a network failure.
var ErrTransient = errors.New("the provider failed for now")

// Transient marks a failure to reach the provider, such as a network error, as
// one that may pass.
func Transient(call string, err error) error {
	return fmt.Errorf("%s: %w: %w", call, ErrTransient, err)
}

// Rejects reports whether an HTTP status says the key itself was refused.
func Rejects(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

// mayPass reports whether an HTTP status says the failure may pass on a retry.
func mayPass(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

// StatusError describes a failed call, wrapping ErrKeyRejected when the status
// says the key was refused, so a session can tell the player, and ErrTransient
// when it may pass, so a block can retry.
func StatusError(call string, resp *http.Response, detail string) error {
	switch {
	case Rejects(resp.StatusCode):
		return fmt.Errorf("%s: %s%s: %w", call, resp.Status, detail, ErrKeyRejected)
	case mayPass(resp.StatusCode):
		return fmt.Errorf("%s: %s%s: %w", call, resp.Status, detail, ErrTransient)
	}
	return fmt.Errorf("%s: %s%s", call, resp.Status, detail)
}

// checkClient is short on patience: a check runs while a player waits.
var checkClient = &http.Client{Timeout: 10 * time.Second}

// CheckKey asks a provider's url with the key, a free request such as listing
// models. It answers nil, ErrKeyRejected or ErrUnreachable, wrapped.
func CheckKey(ctx context.Context, keys KeyFunc, url string) error {
	key, err := keys(ctx)
	if err != nil {
		return fmt.Errorf("resolve api key: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := checkClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case Rejects(resp.StatusCode):
		return fmt.Errorf("%w: %s", ErrKeyRejected, resp.Status)
	default:
		return fmt.Errorf("%w: %s", ErrUnreachable, resp.Status)
	}
}
