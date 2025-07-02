package events

import "time"

const (
	TypeLoginEvent = "LOGIN_EVENT"
)

type LoginEvent struct {
	Timestamp time.Time
	User      string
	Domain    string
	Hash      string
	DeviceID  string
}
