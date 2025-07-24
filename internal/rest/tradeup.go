package rest

import (
	"context"

	"github.com/erobx/csupgrade-go-api/internal/model"
	"github.com/erobx/csupgrade-go-api/internal/tradeup"
	"github.com/gofiber/fiber/v2"
)

type TradeupHandler struct {
	Service *tradeup.Service
}

func NewTradeupHandler(service *tradeup.Service) *TradeupHandler {
	return &TradeupHandler{Service: service}
}

func (h *TradeupHandler) AddItem(c *fiber.Ctx) error {
	var req struct {
		TradeupID string 	`json:"tradeup_id"`
		Item model.Item 	`json:"item"`	
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	err := h.Service.AddItemToTradeup(context.Background(), req.TradeupID, req.Item.UserID, req.Item)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not add item")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
