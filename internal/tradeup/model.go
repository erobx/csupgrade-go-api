package tradeup

type Item struct {
	ID		string 	`json:"id"`
	Name	string 	`json:"name"`
	Value	float64 `json:"value"`
	UserID	string 	`json:"user_id"`
}

type Tradeup struct {
	ID			string 	`json:"id"`
	Items		[]Item 	`json:"items"`
	Capacity 	int 	`json:"capacity"`
}

func (t *Tradeup) IsFull() bool {
	return len(t.Items) >= t.Capacity
}
