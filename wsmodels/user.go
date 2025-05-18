package wsmodels

type User struct {
	Id         string
	Name       string
	ProfileUrl string

	Status   Status
	Meta     map[string]interface{}
	Joined   int64
	LastSeen int64
}

type Status string

const (
	StatusConnected    Status = "connected"
	StatusDisconnected Status = "disconnected"
	StatusError        Status = "error"
	StatusIdle         Status = "idle"
)
