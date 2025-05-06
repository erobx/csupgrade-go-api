package app

import (
	"log"
	"strings"
	"time"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-jwt/jwt/v5"
)

func (s *Server) UseMiddleware() {
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://csupgrade.ebob.dev, http://localhost:5173, http://10.0.0.28:5173",
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		ExposeHeaders:    "X-New-Token",
	}))

	s.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
}

func (s *Server) Protect() {
	s.app.Use(jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: jwtware.RS256,
			Key:    s.privateKey.Public(),
		},
		ErrorHandler: s.InvalidJWT(),
	}))
}

func (s *Server) InvalidJWT() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Only handle JWT errors
		if err == nil || !strings.Contains(err.Error(), "token is expired") {
			return fiber.ErrUnauthorized
		}

		// Get the expired token from header (or cookie if you use that)
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.ErrUnauthorized
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return s.privateKey.Public(), nil
		})

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return fiber.ErrUnauthorized
		}

		// Re-issue new token
		newTokenStr, err := s.issueNewToken(claims)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		// Return new token in response header or body
		c.Set("X-New-Token", newTokenStr)
		log.Printf("issued new token for %s\n", claims["id"].(string))

		// Optional: continue to next middleware/handler
		// return c.Next()
		// But Fiber JWT middleware doesn't allow continuing after error, so:
		return c.Status(fiber.StatusUnauthorized).SendString("Token expired, issued new token")
	}
}

func (s *Server) issueNewToken(claims jwt.MapClaims) (string, error) {
	newClaims := jwt.MapClaims{
		"id":                  claims["id"],
		"email":               claims["email"],
		"refreshTokenVersion": claims["refreshTokenVersion"],
		"exp":                 time.Now().Add(23 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, newClaims)
	return token.SignedString(s.privateKey)
}
