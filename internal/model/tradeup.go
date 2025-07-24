package model

type Tradeup struct {
	ID		string `json:"id"`
	Rarity 	string `json:"rarity"`
	Capacity int	`json:"capacity"`
	Items 	[]Item	`json:"items"`
}

func (t *Tradeup) IsFull() bool {
	return len(t.Items) >= t.Capacity
}
