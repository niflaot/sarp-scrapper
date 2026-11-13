package presence

// Character is one entry of the public online-players payload.
type Character struct {
	ID          int64  `json:"character_id"`
	Name        string `json:"character_name"`
	Level       int    `json:"character_level"`
	AccountName string `json:"account_name"`
}

type onlineResponse struct {
	Payload []Character `json:"payload"`
}
