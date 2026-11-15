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

// Account is one entry of the home online-users payload.
type Account struct {
	ID         int64  `json:"accountId"`
	Name       string `json:"accountName"`
	AdminLevel int    `json:"adminLevel"`
}

type onlineAccountsResponse struct {
	Payload *struct {
		Users []Account `json:"users"`
	} `json:"payload"`
}
