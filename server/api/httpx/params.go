package httpx

import (
	"net/http"

	"github.com/google/uuid"
)

// PathParamUUID parses a path parameter as UUID using Go 1.22+ r.PathValue
func PathParamUUID(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue(name))
}

// PathParam returns a path parameter value using Go 1.22+ r.PathValue
func PathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

// QueryParam returns a query parameter value
func QueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}
