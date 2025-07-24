package model

type Item struct {
    InvID   int     `json:"invId"`
	UserID	string	`json:"userId"`
    Data    any     `json:"data"`
    Visible bool    `json:"visible"`
}
