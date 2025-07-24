package model

type Item struct {
    InvID   int     `json:"invId"`
    Data    any     `json:"data"`
    Visible bool    `json:"visible"`
}
