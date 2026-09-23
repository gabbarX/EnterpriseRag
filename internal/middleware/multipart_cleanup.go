package middleware

import "github.com/gin-gonic/gin"

// MultipartFormCleanup deletes the files that multipart parsing wrote into the
// system temp directory once the request has finished. It takes the Gin request
// context c and returns nothing; requests that never parsed a multipart form,
// or that stayed entirely in memory, delete nothing. Register this middleware
// ahead of all routes so it covers successful, failed and panic-recovered
// uploads.
func MultipartFormCleanup() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// Go sets MultipartForm once ParseMultipartForm has spilled to disk.
			// RemoveAll only deletes the multipart temp files it created itself and
			// does not touch business files that have already been persisted.
			if c.Request != nil && c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		c.Next()
	}
}
