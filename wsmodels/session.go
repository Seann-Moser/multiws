package wsmodels

type Session struct {
	Users        []User                 `json:"users"`
	ID           string                 `json:"id"`
	History      []Event                `json:"history"`
	MaxHistory   int                    `json:"maxHistory"`
	IdleDuration int                    `json:"idleDuration"`
	Meta         map[string]interface{} `json:"meta"`
	Self         User                   `json:"-"`
}
