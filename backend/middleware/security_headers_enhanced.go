package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds comprehensive security headers to responses
// This does NOT change any business logic, just adds HTTP headers
func SecurityHeadersMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy - Prevents XSS attacks
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' "+frontendURL+"; frame-ancestors 'none'")

		// X-Content-Type-Options - Prevents MIME sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options - Prevents clickjacking
		c.Header("X-Frame-Options", "DENY")

		// X-XSS-Protection - Legacy XSS protection for older browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy - Controls referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// Strict-Transport-Security - Forces HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Don't cache sensitive responses
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// CORSSecureMiddleware adds secure CORS headers
// Prevents unauthorized cross-origin requests
func CORSSecureMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is in allowed list
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With,Accept,Origin")
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
			c.Header("Access-Control-Expose-Headers", "Content-Length,X-JSON-Response-Body")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if isAllowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		c.Next()
	}
}
