package handler

import (
	"cgl/db"
	"cgl/obj"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Handler func(request Request) (interface{}, *obj.HTTPError)

type Request struct {
	R          *http.Request
	User       *obj.User
	Ctx        context.Context
	PathParams map[string]string
}

// GetParam returns a query parameter value
func (r *Request) GetParam(key string) string {
	return r.R.URL.Query().Get(key)
}

// GetPathParam returns a path parameter value (e.g., "id" from "/games/{id}")
func (r *Request) GetPathParam(key string) string {
	if r.PathParams == nil {
		return ""
	}
	return r.PathParams[key]
}

// GetPathParamUUID returns a path parameter as UUID
func (r *Request) GetPathParamUUID(key string) (uuid.UUID, error) {
	return uuid.Parse(r.GetPathParam(key))
}

// IsAdmin returns true if the current user has admin role
func (r *Request) IsAdmin() bool {
	return r.User != nil && r.User.Role.Role == obj.RoleAdmin
}

// RequireAdmin returns an error if the current user is not an admin
func (r *Request) RequireAdmin() *obj.HTTPError {
	if !r.IsAdmin() {
		return &obj.HTTPError{StatusCode: 403, Message: "Forbidden: admin access required"}
	}
	return nil
}

// Body returns the request body as bytes
func (r *Request) Body() ([]byte, *obj.HTTPError) {
	body, err := io.ReadAll(r.R.Body)
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 400, Message: "Failed to read body: " + err.Error()}
	}
	return body, nil
}

// BodyJSON decodes the request body as JSON into the given struct
func (r *Request) BodyJSON(v any) *obj.HTTPError {
	if err := json.NewDecoder(r.R.Body).Decode(v); err != nil {
		return &obj.HTTPError{StatusCode: 400, Message: "Invalid JSON: " + err.Error()}
	}
	return nil
}

// BodyYAML decodes the request body as YAML into the given struct
func (r *Request) BodyYAML(v any) *obj.HTTPError {
	body, httpErr := r.Body()
	if httpErr != nil {
		return httpErr
	}
	if err := yaml.Unmarshal(body, v); err != nil {
		return &obj.HTTPError{StatusCode: 400, Message: "Invalid YAML: " + err.Error()}
	}
	return nil
}

type Endpoint struct {
	// The path pattern of the endpoint (e.g., "/api/games/{id}/yaml")
	Path string
	// pathRegex is compiled from Path for matching
	pathRegex *regexp.Regexp
	// paramNames extracted from Path (e.g., ["id"])
	paramNames     []string
	Public         bool
	RequiredScopes []string
	ContentType    string
	Handler        http.HandlerFunc
}

// Type patterns for path parameters
var typePatterns = map[string]string{
	"uuid": `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	"int":  `[0-9]+`,
}

// compilePath converts a path pattern like "/api/games/{id:uuid}/yaml" into a regex
// and extracts parameter names. Supports typed parameters: {name:type}
// Supported types: uuid, int. Default (no type) matches any non-slash characters.
func compilePath(pattern string) (*regexp.Regexp, []string) {
	var paramNames []string
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"

	// Find all {param} or {param:type} patterns and replace with capture groups
	paramRegex := regexp.MustCompile(`\\\{([^}:]+)(?::([^}]+))?\\\}`)
	matches := paramRegex.FindAllStringSubmatch(regexPattern, -1)
	for _, match := range matches {
		paramNames = append(paramNames, match[1]) // Just the name, not the type
	}

	// Replace each placeholder with appropriate regex
	regexPattern = paramRegex.ReplaceAllStringFunc(regexPattern, func(match string) string {
		// Parse the match to get type
		parts := paramRegex.FindStringSubmatch(match)
		if len(parts) >= 3 && parts[2] != "" {
			if typePattern, ok := typePatterns[parts[2]]; ok {
				return "(" + typePattern + ")"
			}
		}
		// Default: match any non-slash characters
		return `([^/]+)`
	})

	return regexp.MustCompile(regexPattern), paramNames
}

// extractPathParams extracts parameter values from a URL path using the compiled regex
func (e *Endpoint) extractPathParams(urlPath string) map[string]string {
	params := make(map[string]string)
	if e.pathRegex == nil {
		return params
	}

	matches := e.pathRegex.FindStringSubmatch(urlPath)
	if len(matches) > 1 {
		for i, name := range e.paramNames {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}
	}
	return params
}

// MatchesPath checks if a URL path matches this endpoint's pattern
func (e *Endpoint) MatchesPath(urlPath string) bool {
	if e.pathRegex == nil {
		// Fallback to prefix matching for legacy endpoints
		return strings.HasPrefix(urlPath, e.Path)
	}
	return e.pathRegex.MatchString(urlPath)
}

func NewEndpoint(path string, public bool, contentType string, endpointHandler Handler) Endpoint {
	pathRegex, paramNames := compilePath(path)
	endpoint := Endpoint{
		Path:           path,
		pathRegex:      pathRegex,
		paramNames:     paramNames,
		Public:         public,
		RequiredScopes: []string{},
		ContentType:    contentType,
	}

	endpoint.Handler = func(w http.ResponseWriter, r *http.Request) {
		var httpError *obj.HTTPError
		var err error

		request := Request{
			R:          r,
			Ctx:        context.Background(),
			PathParams: endpoint.extractPathParams(r.URL.Path),
		}

		SetCorsHeaders(w)
		SetNoCacheHeaders(w)
		w.Header().Set("Content-Type", endpoint.ContentType)

		log.Printf("Handling request for %s", r.URL.Path)

		// Check for CGL JWT user ID first
		if userId, ok := r.Context().Value(CglUserIdKey{}).(string); ok && userId != "" {
			request.User, err = db.GetUserByID(request.Ctx, uuid.MustParse(userId))
			if err != nil {
				httpError = &obj.HTTPError{StatusCode: http.StatusUnauthorized, Message: "User not found"}
			}
			log.Printf("valid jwt user: %s (%s) for %s %s", userId, request.User.Name, r.Method, r.URL.Path)
		} else if tokenObj := r.Context().Value(jwtmiddleware.ContextKey{}); tokenObj != nil {
			// Fall back to Auth0 token
			token := tokenObj.(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*CustomClaims)
			for _, requiredScope := range endpoint.RequiredScopes {
				if !claims.HasScope(requiredScope) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"message":"Insufficient scope."}`))
					return
				}
			}

			if auth0ID := token.RegisteredClaims.Subject; auth0ID != "" {
				request.User, err = db.GetUserByAuth0ID(request.Ctx, auth0ID)

				// unknown user - create them
				if err != nil {
					request.User, err = db.CreateUser(request.Ctx, "Unnamed Auth0 User", nil, auth0ID)
					if err != nil {
						httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to create user"}
					}
				}
			}
		}

		if !public && request.User == nil {
			httpError = &obj.HTTPError{StatusCode: http.StatusUnauthorized, Message: "Unauthorized"}
		}

		var res interface{}
		if httpError == nil {
			log.Printf("Passing over to handler")
			res, httpError = endpointHandler(request)
		}

		var resBytes []byte
		if httpError == nil {
			switch endpoint.ContentType {
			case "application/json":
				if resBytes, err = json.Marshal(res); err != nil {
					httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed to marshal json"}
				}
			case "image/png":
				resBytes = res.([]byte)
			case "text/csv":
				resBytes = []byte(res.(string))
			default:
				httpError = &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Handler has unknown content type"}
			}
		}

		if httpError != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(httpError.StatusCode)
			_, _ = w.Write(httpError.Json())
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resBytes)
	}

	return endpoint
}
