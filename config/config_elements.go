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
	WebSocket *websocket.Conn
}

type FormElement struct {
	Id string `json:"id"`
	CreatedAt time.Time
	EditedBy ClientObject
	Question string `json:"question"`
}
type ClientObject struct {
	IPAddress string `json:"ipAddress,omitempty"'`
	Username string `json:"username,omitempty"`
	UserId string `json:"userId,omitempty"`
	Colour string `json:"colour,omitempty"`
	ClientWebSocket *websocket.Conn
}
