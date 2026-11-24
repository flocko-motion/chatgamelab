package adapters

import (
	"encoding/base64"
	"net/http"
)

// DataURL carries a picture inline, as providers take one in a request.
func DataURL(image []byte) string {
	return "data:" + http.DetectContentType(image) + ";base64," + base64.StdEncoding.EncodeToString(image)
}
