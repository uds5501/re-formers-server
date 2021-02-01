package config

import "time"

type SampleElement struct {
	Message string `json:"message"`
	MessageType string `json:"messageType"`
}

type FormElement struct {
	Id string `json:"id"`
	CreatedAt time.Time
	EditedBy ClientElement
	Question string `json:"question"`
}
type ClientElement struct {
	Name string `json:"name"`
}