// package: obj / core domain data model
// type:    data
// job:     content hash for generated images, shared by the image cache, the DB
//          layer and the SSE stream so the ?v=<hash> URL and the ETag agree.
// limits:  identity/change-detection only; not a cryptographic guarantee.
package obj

import (
	"crypto/md5"
	"encoding/hex"
)

// ImageHash returns a short, stable content hash for image bytes. Empty input
// yields an empty string. 16 hex chars (first 8 bytes of MD5) is plenty to
// distinguish one generated image from another.
func ImageHash(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:8])
}
