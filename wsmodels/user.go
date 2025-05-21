package wsmodels

type User struct {
	Id         string
	Name       string
	ProfileUrl string

	Status   string
	Meta     map[string]interface{}
	Joined   int64
	LastSeen int64
	Host     bool
}

const (
	StatusConnected    string = "connected"
	StatusDisconnected string = "disconnected"
	StatusError        string = "error"
	StatusIdle         string = "idle"
)
