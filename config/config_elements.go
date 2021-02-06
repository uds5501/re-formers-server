package config

import (
	"github.com/gorilla/websocket"
	"time"
)

type SampleElement struct {
	Message string `json:"message"`
	MessageType string `json:"messageType"`
}

type ClientRequest struct {
	MessageType string `json:"messageType"`
	UserId string `json:"userId,omitempty"`
	EntryToken string `json:"entryToken,omitempty"`
	WebSocket *websocket.Conn
}
type ServerClientCommunication struct {
	MessageType string `json:"messageType"`
	ClientObject *ClientObject `json:"clientObject"`
}

type FormElement struct {
	Id string `json:"id"`
	CreatedAt time.Time
	EditedBy ClientObject
	Question string `json:"question"`
}
type ClientObject struct {
	IPAddress string `json:"ipAddress,omitempty"'`
	Username string `json:"userName,omitempty"`
	EntryToken string `json:"entryToken,omitempty"`
	Colour string `json:"colour,omitempty"`
	ClientWebSocket *websocket.Conn
}

type PeriodicUpdater struct {
	ClientData []*ClientObject `json:"clientList,omitempty"`
	FormData []FormElement `json:formlist,omitempty`
	MessageType string `json:messageType,omitempty`
}