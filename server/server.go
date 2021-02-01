package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/uds5501/re-formers-server/config"
	"github.com/uds5501/re-formers-server/utils"

)

type WebsocketServer struct {
	clients map[*config.ClientObject]bool
	broadcast chan config.FormElement
	requestUpgrader websocket.Upgrader
	Util *utils.Utils
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan config.FormElement)


func (wss *WebsocketServer) HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func (wss *WebsocketServer) handleMessages() {
	for {
		newElement := <- wss.broadcast
		for client := range wss.clients {
			err := client.ClientWebSocket.WriteJSON(newElement)
			if err != nil {
				log.Printf("error: %v", err)
				client.ClientWebSocket.Close()
				delete(wss.clients, client)
			}
		}
	}
}

func (wss *WebsocketServer) wsEndPoint(w http.ResponseWriter, r *http.Request) {
	wss.requestUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
	if wss.Util.AllowEntry() == false {
		wss.requestUpgrader.Error(w, r, 403, errors.New("Room is full"))
		return
	}
	ws, err := wss.requestUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
	}
	log.Println("Client successfully Connected! ")
	defer ws.Close()
	// change this crap flow
	//wss.clients[ws] = true
	for {
		var newElement config.ClientRequest
		mType, p, err := ws.ReadMessage()
		err = json.Unmarshal(p, &newElement)
		newElement.WebSocket = ws
		log.Println("Got client message: ", newElement, mType)
		err = wss.HandleClientMessage(newElement)
		if err != nil {
			log.Fatal("Error occured: ",err)
			//delete(wss.clients, ws)
			break
		}
		//broadcast <- newElement
	}
}

func (wss *WebsocketServer) HandleClientMessage(clientData config.ClientRequest) error{
	if clientData.MessageType == "room entry" {
		userName, colour := wss.Util.AssignData()
		if userName != "-1" && colour != "-1" {
			clientObject := &config.ClientObject{
				Username: userName,
				Colour: colour,
				ClientWebSocket: clientData.WebSocket,
			}
			wss.clients[clientObject] = true
		}
	}
	return nil
}
func (wss *WebsocketServer) SetupServer() {
	http.HandleFunc("/", wss.HomePage)
	http.HandleFunc("/ws", wss.wsEndPoint)
	go wss.handleMessages()
}

func Init() *WebsocketServer{
	var upgrader = websocket.Upgrader {
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
	}
	currentUtility := utils.Init()
	return &WebsocketServer{
		clients: map[*config.ClientObject]bool{},
		broadcast: make(chan config.FormElement),
		requestUpgrader: upgrader,
		Util: currentUtility,
	}
}