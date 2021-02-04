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
	clientTokenMap map[string]*config.ClientObject
	broadcast chan config.FormElement
	requestUpgrader websocket.Upgrader
	Util *utils.Utils
}


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

func (wss *WebsocketServer) chuckClient(object *config.ClientObject) {
	delete(wss.clients, object)
	delete(wss.clientTokenMap, object.EntryToken)
	delete(wss.Util.NameMapper, object.Username+object.Colour)
	log.Println(wss.Util.NameMapper)
}

func (wss *WebsocketServer) wsEndPoint(w http.ResponseWriter, r *http.Request) {
	wss.requestUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := wss.requestUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
	}
	log.Println("Client successfully Connected! ")
	defer func() {
		log.Println("In defer close")
		ws.Close()
	}()
	// change this crap flow
	//wss.clients[ws] = true
	for {
		var newElement config.ClientRequest
		mType, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("Client Disconnected: ",err)
			//delete(wss.clients, ws)
			break
		}
		err = json.Unmarshal(p, &newElement)
		newElement.WebSocket = ws
		log.Println("Got client message: ", newElement, mType, ws.RemoteAddr().String())
		err2, retrievedClient := wss.HandleClientMessage(newElement)
		if err2 != nil {
			log.Println("Error occured: ",err)
			ws.Close()
			if retrievedClient != nil {
				wss.chuckClient(retrievedClient)

			}
			break
		}
	}
}


func (wss *WebsocketServer) HandleClientMessage(clientData config.ClientRequest) (error, *config.ClientObject){
	if clientData.MessageType == "room entry" {
		// check if it's already registered
		log.Println("Entry Token is : ", clientData.EntryToken)
		clientObj, found := wss.clientTokenMap[clientData.EntryToken]
		log.Println(clientObj, found)
		if found == true {
			// update mapped client's web socket
			delete(wss.clients, clientObj)
			// send a message that yes you were in already, here are your creds
			clientObj.ClientWebSocket = clientData.WebSocket
			wss.clients[clientObj] = true
			msg := wss.Util.CreateMessage("welcome", clientObj)
			err := clientObj.ClientWebSocket.WriteJSON(msg)
			if err != nil {
				return err, clientObj
			}
		} else {
			// new web socket is in non pre defined token mapping
			fmt.Println("In OR CONDITION")
			if wss.Util.AllowEntry() {
				userName, colour := wss.Util.AssignData()
				entryToken := wss.Util.GetEntryToken(10)
				if userName != "-1" && colour != "-1" {
					clientObject := &config.ClientObject{
						Username: userName,
						Colour: colour,
						ClientWebSocket: clientData.WebSocket,
						IPAddress: clientData.WebSocket.RemoteAddr().String(),
						EntryToken: entryToken,
					}
					log.Println("NEW OBJ MADE:", clientObject)
					// map entryToken to client object
					wss.clientTokenMap[entryToken] = clientObject
					// map clientObject to a boolean true for easy broadcast
					wss.clients[clientObject] = true
					msg := wss.Util.CreateMessage("welcome", clientObject)
					err := clientObject.ClientWebSocket.WriteJSON(msg)
					if err != nil {
						return err, clientObj
					}
				}
			} else {
				// return room is full message and disconnect the socket
				log.Println("Room was full")
				err := errors.New("room was full")
				return err, nil
			}
		}

	}
	return nil, nil
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
		clients: make(map[*config.ClientObject]bool),
		broadcast: make(chan config.FormElement),
		requestUpgrader: upgrader,
		Util: currentUtility,
		clientTokenMap: make(map[string]*config.ClientObject),
	}
}