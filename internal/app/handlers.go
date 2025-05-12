package app

import (
	"log"
	"strconv"
	"time"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *Server) register() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newUserRequest := new(api.NewUserRequest)

		if err := c.BodyParser(newUserRequest); err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		userID, err := s.userService.New(newUserRequest)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		log.Printf("Created new user %s\n", userID)

		user, err := s.userService.GetUser(userID)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

        inv, err := s.userService.GetInventory(userID)
        if err != nil {
            s.logger.Error("failed to load inventory", "user", userID)
            return c.SendStatus(fiber.StatusInternalServerError)
        }

		claims := jwt.MapClaims{
			"id":                  userID,
			"email":               user.Email,
			"refreshTokenVersion": user.RefreshTokenVersion,
			"exp":                 time.Now().Add(time.Hour * 23).Unix(),
		}

		t, err := s.issueNewToken(claims)
		if err != nil {
			log.Printf("token.SignedString: %v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{
			"user": user,
            "inventory": inv,
			"jwt":  t,
		})
	}
}

func (s *Server) login() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newLoginRequest := new(api.NewLoginRequest)

		if err := c.BodyParser(newLoginRequest); err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		user, inv, err := s.userService.Login(newLoginRequest)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		claims := jwt.MapClaims{
			"id":                  user.ID,
			"email":               user.Email,
			"refreshTokenVersion": user.RefreshTokenVersion,
			"exp":                 time.Now().Add(time.Hour * 23).Unix(),
		}

		t, err := s.issueNewToken(claims)
		if err != nil {
			log.Printf("token.SignedString: %v", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{
			"user":      user,
			"inventory": inv,
			"jwt":       t,
		})
	}
}

func (s *Server) getUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		jwtUser := c.Locals("user").(*jwt.Token)
		claims := jwtUser.Claims.(jwt.MapClaims)
		userID := claims["id"].(string)

		user, err := s.userService.GetUser(userID)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{
			"user": user,
		})
	}
}

func (s *Server) getInventory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Query("userId")
		jwtUserID := GetUserIDFromClaims(c)

		err := s.validator.ValidateUserID(userID, jwtUserID)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		log.Printf("Requesting inventory for %s\n", userID)

		inventory, err := s.userService.GetInventory(userID)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(inventory)
	}
}

func (s *Server) getRecentTradeups() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userId")
		log.Println("Getting recent tradeups for:", userID)

		recentTradeups, err := s.userService.GetRecentTradeups(userID)
		if err != nil {
			log.Printf("couldn't get recent tradeups for %s - %v\n", userID, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(recentTradeups)
	}
}

func (s *Server) getUserStats() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Params("userId")
		log.Println("Retrieving stats for:", userID)

		return nil
	}
}

func (s *Server) buyCrate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Query("userId")
		crateID := c.Query("crateId")
		amount, _ := strconv.Atoi(c.Query("amount"))

		jwtUserID := GetUserIDFromClaims(c)

		err := s.validator.ValidateUserID(userID, jwtUserID)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		log.Printf("User %s buying crate %s - %d\n", userID, crateID, amount)
		updatedBalance, addedItems, err := s.storeService.BuyCrate(crateID, userID, amount)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{
			"balance": updatedBalance,
			"items":   addedItems,
		})
	}
}

func (s *Server) addSkinToTradeup() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tradeupID := c.Params("tradeupId")
		invID := c.Query("invId")
		userID := GetUserIDFromClaims(c)

		err := s.tradeupService.AddSkinToTradeup(tradeupID, invID, userID)
		if err != nil {
			if err == api.ErrMaxContribution {
				return c.SendStatus(fiber.StatusBadRequest)
			}

			log.Println(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

func (s *Server) removeSkinFromTradeup() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tradeupID := c.Params("tradeupId")
		invID := c.Query("invId")
		userID := GetUserIDFromClaims(c)

		err := s.tradeupService.RemoveSkinFromTradeup(tradeupID, invID, userID)
		if err != nil {
			log.Println(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

func (s *Server) handleWebSocket(c *websocket.Conn) {
	userID := c.Query("userId")

	sessionID := ""
	if userID == "" {
		sessionID = uuid.NewString()
		userID = sessionID
	}

	client := &Client{
		Conn:          c,
		UserID:        userID,
		SessionID:     sessionID,
		SubscribedAll: false,
		SubscribedID:  "",
	}

	s.logger.Info("New connection", "user", client.UserID)

	s.Lock()
	s.clients[userID] = client
	s.Unlock()

	defer func() {
		s.Lock()
		delete(s.clients, userID)
		s.Unlock()
		c.Close()
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket closed for", userID, ":", err)
			break
		}

		s.handleSubscription(userID, msg)
	}
}
