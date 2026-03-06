package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

		// CSP (safe default)
		c.Header(
			"Content-Security-Policy",
			"default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'",
		)

		c.Next()
	}
}
