package app

import (
	"reflect"

	"github.com/erobx/csupgrade/backend/pkg/api"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func GetUserIDFromClaims(c *fiber.Ctx) string {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["id"].(string)
	return userID
}

func TradeupEqual(t1, t2 api.Tradeup) bool {
	return reflect.DeepEqual(t1, t2)
}

func TradeupSlicesEqual(s1, s2 []api.Tradeup) bool {
	if len(s1) != len(s2) {
		return false
	}

	m1 := make(map[int]api.Tradeup)
	m2 := make(map[int]api.Tradeup)

	for _, t := range s1 {
		m1[t.ID] = t
	}
	for _, t := range s2 {
		m2[t.ID] = t
	}

	for id, t1 := range m1 {
		t2, exists := m2[id]
		if !exists || !TradeupEqual(t1, t2) {
			return false
		}
	}

	return true
}
