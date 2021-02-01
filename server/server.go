package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/uds5501/re-formers-server/config"
)
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan config.FormElement)

var upgrader = websocket.Upgrader {
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}
func handleMessages() {
	for {
		newElement := <- broadcast
		for client := range clients {
			err := client.WriteJSON(newElement)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func wsEndPoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	//log.Println(ws.)
	log.Println("Client successfully Connected! ")

	defer ws.Close()
	clients[ws] = true
	for {
		var newElement config.SampleElement
		mType, p, err := ws.ReadMessage()
		err = json.Unmarshal(p, &newElement)
		//json.Unmarshal(p, &newElement)
		//err := ws.ReadJSON(&newElement)
		log.Println("Got data", newElement, mType)
		if err != nil {
			log.Fatal("Error occured: ",err)
			delete(clients, ws)
			break
		}
		//broadcast <- newElement
	}
}

func SetupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndPoint)
	go handleMessages()
}