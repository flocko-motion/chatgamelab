// package: testutil / test assertion helpers
// type:    logic
// job:     Must*/Fail* UserClient wrappers and standalone assertion/debug helpers
// limits:  assertions and error-expectation wrappers only; no request transport (-> testclient)
package testutil

import (
	"encoding/json"
	"fmt"
	"testing"
)

// MustGet performs GET and fails test on error
func (u *UserClient) MustGet(endpoint string, out interface{}) {
	u.t.Helper()
	if err := u.Get(endpoint, out); err != nil {
		u.t.Fatalf("User %q GET %s failed: %v", u.Name, endpoint, err)
	}
}

// MustPost performs POST and fails test on error
func (u *UserClient) MustPost(endpoint string, payload interface{}, out interface{}) {
	u.t.Helper()
	if err := u.Post(endpoint, payload, out); err != nil {
		u.t.Fatalf("User %q POST %s failed: %v", u.Name, endpoint, err)
	}
}

// MustPatch performs PATCH and fails test on error
func (u *UserClient) MustPatch(endpoint string, payload interface{}, out interface{}) {
	u.t.Helper()
	if err := u.Patch(endpoint, payload, out); err != nil {
		u.t.Fatalf("User %q PATCH %s failed: %v", u.Name, endpoint, err)
	}
}

// MustDelete performs DELETE and fails test on error
func (u *UserClient) MustDelete(endpoint string) {
	u.t.Helper()
	if err := u.Delete(endpoint); err != nil {
		u.t.Fatalf("User %q DELETE %s failed: %v", u.Name, endpoint, err)
	}
}

// FailGet expects GET to fail and validates the error
func (u *UserClient) FailGet(endpoint string, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Get(endpoint, nil)
	if err == nil {
		u.t.Fatalf("User %q GET %s: expected error but got none", u.Name, endpoint)
	}

	if len(validators) == 0 {
		u.t.Logf("User %q GET %s: got expected error: %v", u.Name, endpoint, err)
		return
	}

	for _, validator := range validators {
		if !validator(err) {
			u.t.Fatalf("User %q GET %s: error validation failed: %v", u.Name, endpoint, err)
		}
	}
	u.t.Logf("User %q GET %s: got expected error: %v", u.Name, endpoint, err)
}

// FailPost expects POST to fail and validates the error
func (u *UserClient) FailPost(endpoint string, payload interface{}, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Post(endpoint, payload, nil)
	validateError(u.t, err, fmt.Sprintf("User %q POST %s", u.Name, endpoint), validators...)
}

// FailPatch expects PATCH to fail and validates the error
func (u *UserClient) FailPatch(endpoint string, payload interface{}, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Patch(endpoint, payload, nil)
	validateError(u.t, err, fmt.Sprintf("User %q PATCH %s", u.Name, endpoint), validators...)
}

// FailDelete expects DELETE to fail and validates the error
func (u *UserClient) FailDelete(endpoint string, validators ...ErrorValidator) {
	u.t.Helper()
	err := u.Delete(endpoint)
	validateError(u.t, err, fmt.Sprintf("User %q DELETE %s", u.Name, endpoint), validators...)
}

// PrintJSON prints a value as formatted JSON for debugging
//
//deadcode:keep // ad-hoc debugging helper for tests
func PrintJSON(t *testing.T, label string, v interface{}) {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Logf("%s: (marshal error: %v)", label, err)
		return
	}
	t.Logf("%s: %s", label, string(data))
}

// AssertEqual checks if two values are equal
//
//deadcode:keep // part of the shared test-assertion helper API
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()

	if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEmpty checks if a value is not empty
//
//deadcode:keep // part of the shared test-assertion helper API
func AssertNotEmpty(t *testing.T, value interface{}, msg string) {
	t.Helper()

	if fmt.Sprintf("%v", value) == "" || fmt.Sprintf("%v", value) == "<nil>" {
		t.Errorf("%s: value is empty", msg)
	}
}
