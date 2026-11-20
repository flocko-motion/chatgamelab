// package: api / schemaf endpoints
// type:    logic
// job:     tells someone choosing a name whether it is free
// limits:  name storage lives in db; the profanity list lives in constants
package api

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"cgl/constants"
	"cgl/db"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// CheckNameAvailabilityEndpoint checks whether a name is free to register.
//
// A name longer than 24 characters is unavailable; a profane one is
// unavailable and flagged profane.
type CheckNameAvailabilityEndpoint struct{}

func (CheckNameAvailabilityEndpoint) Method() string             { return "GET" }
func (CheckNameAvailabilityEndpoint) Path() string               { return "/api/auth/check-name" }
func (CheckNameAvailabilityEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (CheckNameAvailabilityEndpoint) Handle(ctx context.Context, req CheckNameReq) (CheckNameResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CheckNameResp{}, errBadRequest("Name parameter is required")
	}
	if utf8.RuneCountInString(name) > 24 {
		return CheckNameResp{Available: false}, nil
	}
	if constants.IsProfane(name) {
		return CheckNameResp{Available: false, Profane: true}, nil
	}
	taken, err := db.IsNameTaken(ctx, name)
	if err != nil {
		return CheckNameResp{}, fmt.Errorf("failed to check name availability: %w", err)
	}
	return CheckNameResp{Available: !taken}, nil
}

// CheckNameReq is the name to check.
type CheckNameReq struct {
	Name string `query:"name" json:"-"`
}

// CheckNameResp says whether the name is free, and whether it is profane.
type CheckNameResp struct {
	Available bool `json:"available"`
	Profane   bool `json:"profane,omitempty"`
}
