package mcpsdk

type EventType string

const (
	EventTypeMigrate    EventType = "migrate"
	EventTypeKickout    EventType = "kickout"
	EventTypeDisconnect EventType = "disconnect"
)
