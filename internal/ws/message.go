package ws

import "encoding/json"

type SubscribeRequest struct {
	UserID		string	`json:"userId"`
	TradeupID 	int		`json:"tradeupId"`
}

type UnsubscribeRequest struct {
	UserID		string	`json:"userId"`
	TradeupID 	int		`json:"tradeupId"`
}

// from client -> server
type ClientMessageType int

const (
	AddItem ClientMessageType = iota
	RemoveItem
	SubscribeTradeup
	UnsubscribeTradeup
)

type ClientMessage struct {
	UserID	string				`json:"userId"`
	Type	ClientMessageType	`json:"type"`
	Payload	json.RawMessage		`json:"payload"`
}

func (c ClientMessage) Encode() []byte {
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return jsonBytes
}

type AddItemPayload struct {
	TradeupID	int		`json:"tradeupId"`
	ItemID		int		`json:"itemId"`
}

type RemoveItemPayload struct {
	TradeupID	int		`json:"tradeupId"`
	ItemID		int		`json:"itemId"`
}

type SubscribePayload struct {
	TradeupID	int		`json:"tradeupId"`
}

// from server -> client
type ServerMessageType int

const (
	ItemAdded ServerMessageType = iota
	ItemRemoved
	Notification
	Error
	TradeupUpdate
	TradeupFinished
	SubscribedTradeup
)

type ServerMessage struct {
	Type 	ServerMessageType 	`json:"type"`
	Payload json.RawMessage 	`json:"payload"`
}

func (s ServerMessage) Encode() []byte {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return jsonBytes
}

type ItemAddedPayload struct {
	TradeupID 	int		`json:"tradeupId"`
	ItemID 		int		`json:"itemId"`
}
