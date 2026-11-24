package adapters

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrKeyRejected is a provider refusing the key, as opposed to failing to answer.
var ErrKeyRejected = errors.New("the provider rejected the key")

// ErrUnreachable is a provider that could not be asked, or failed to answer.
var ErrUnreachable = errors.New("the provider is unreachable")

// ErrNoCredit is a provider refusing the call because the account behind the
// key has no credit left: the key is fine, but nothing runs until it is
// topped up. It does not pass on a retry.
var ErrNoCredit = errors.New("the account behind the key has no credit left")

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

// noCreditMarks are what a provider's refusal says when the account has no
// credit left: OpenAI's error type and code.
var noCreditMarks = []string{"insufficient_quota", "credit_balance_exhausted"}

// noCredit reports whether a refusal says the account has no credit left:
// 402 Payment Required, or a 429 whose body names the credit as used up. A
// plain 429 is a rate limit, which passes.
func noCredit(status int, body string) bool {
	if status == http.StatusPaymentRequired {
		return true
	}
	if status != http.StatusTooManyRequests {
		return false
	}
	for _, mark := range noCreditMarks {
		if strings.Contains(body, mark) {
			return true
		}
	}
	return false
}

// bodyLimit is how much of a refusal's body is read to tell why.
const bodyLimit = 4 << 10

// StatusError describes a failed call, wrapping ErrKeyRejected when the status
// says the key was refused and ErrNoCredit when the account has no credit
// left, so a session can tell the player, and ErrTransient when it may pass,
// so a block can retry.
func StatusError(call string, resp *http.Response, detail string) error {
	var body string
	if resp.Body != nil && (resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests) {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
		body = string(raw)
	}
	switch {
	// A caller that read the body for the detail has it there instead.
	case noCredit(resp.StatusCode, body+detail):
		return fmt.Errorf("%s: %s%s: %w", call, resp.Status, detail, ErrNoCredit)
	case Rejects(resp.StatusCode):
		return fmt.Errorf("%s: %s%s: %w", call, resp.Status, detail, ErrKeyRejected)
	case mayPass(resp.StatusCode):
		return fmt.Errorf("%s: %s%s: %w", call, resp.Status, detail, ErrTransient)
	}
	return fmt.Errorf("%s: %s%s", call, resp.Status, detail)
}

// ProbePrompt is what a key check asks: a question a model answers in a word,
// so the call costs next to nothing.
const ProbePrompt = "1+1="
