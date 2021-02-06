package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/uds5501/re-formers-server/config"
	"github.com/uds5501/re-formers-server/utils"
)

type WebsocketServer struct {
	clients map[*config.ClientObject]bool
	clientTokenMap map[string]*config.ClientObject
	broadcast chan config.FormElement
	clientRoomActivity chan string
	requestUpgrader websocket.Upgrader
	Util *utils.Utils
	userTicker *time.Ticker
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

func (wss *WebsocketServer) handleCustomMessages() {
	for {
		newMessage := <- wss.clientRoomActivity
		log.Println("Message to distribute: ", newMessage)
		for client := range wss.clients {
			fmt.Println("SENDING MESSAGE TO ", client)
			err := client.ClientWebSocket.WriteMessage(1, []byte(newMessage))
			if err != nil {
				log.Println("Error in sending ", newMessage, "to ", client )
			}
		}
	}
}

func (wss *WebsocketServer) chuckClient(object *config.ClientObject) {
	delete(wss.clients, object)
	delete(wss.clientTokenMap, object.EntryToken)
	delete(wss.Util.NameMapper, object.Username+object.Colour)
}

func (wss *WebsocketServer) wsEndPoint(w http.ResponseWriter, r *http.Request) {
	wss.requestUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := wss.requestUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}
	var preListen *config.ClientObject
	defer func() {
		var e error
		if preListen != nil {
			e = preListen.ClientWebSocket.Close()
			delete(wss.clients, preListen)
		}
		fmt.Println(e)
	}()
	for {
		var newElement config.ClientRequest
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("Client Disconnected: ",err)
			//delete(wss.clients, ws)
			break
		}
		err = json.Unmarshal(p, &newElement)
		newElement.WebSocket = ws
		err2, retrievedClient := wss.HandleClientMessage(newElement)
		preListen = retrievedClient
		if err2 != nil {
			log.Println("Error occured: ",err2)
			if retrievedClient != nil {
				wss.chuckClient(retrievedClient)
			}
			break
		}
	}
}


func (wss *WebsocketServer) HandleClientMessage(clientData config.ClientRequest) (error, *config.ClientObject){
	log.Println("client data: ", clientData)
	if clientData.MessageType == "room entry" {
		// check if it's already registered
		log.Println("Entry Token is : ", clientData.EntryToken)
		clientObj, found := wss.clientTokenMap[clientData.EntryToken]
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
			} else {
				// gonna think about this i guess
				wss.clientRoomActivity <- fmt.Sprintf(`{"messageType": "user-joined", "userName": "%s", "userColor": "%s"}`, clientObj.Username, clientObj.Colour)
			}
		} else {
			// new web socket is in non pre defined token mapping
			log.Println("we could not find you")
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
					// map entryToken to client object
					wss.clientTokenMap[entryToken] = clientObject
					// map clientObject to a boolean true for easy broadcast
					wss.clients[clientObject] = true
					msg := wss.Util.CreateMessage("welcome", clientObject)
					err := clientObject.ClientWebSocket.WriteJSON(msg)
					if err != nil {
						return err, clientObject
					} else {
						wss.clientRoomActivity <- fmt.Sprintf(`{"messageType": "user-joined", "userName": "%s", "userColor": "%s"}`, clientObject.Username, clientObject.Colour)
					}
				}
			} else {
				// return room is full message and disconnect the socket
				log.Println("Room was full")
				msg := wss.Util.CreateMessage("room-full", nil)
				err := clientData.WebSocket.WriteJSON(msg)
				err = errors.New("room was full")
				return err, nil
			}
		}
	}
	return nil, nil
}

func (wss *WebsocketServer) handleRoomExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(r.Body)
	var cr config.ClientRequest
	err := decoder.Decode(&cr)
	if err != nil {
		fmt.Println(err)
	}
	clientObj, found := wss.clientTokenMap[cr.EntryToken]
	if found {
		err = clientObj.ClientWebSocket.Close()
		if err != nil {
			log.Fatal(err)
		}
		wss.chuckClient(clientObj)
		log.Println("Logged out ", clientObj.Username)
		//jData := json.Marshal()
		w.Write([]byte(fmt.Sprintf("`{'message': '%s''}`", "logged-out")))
	}
}

func (wss *WebsocketServer) roomUpdater() {
	for {
		select {
		case t := <- wss.userTicker.C:
			fmt.Println("Sending elements at", t)
			updater := config.PeriodicUpdater{[]*config.ClientObject{}, []config.FormElement{}, "updater"}
			for clObj := range wss.clients{
				updater.ClientData = append(updater.ClientData, clObj)
			}
			for client := range wss.clients {
				err := client.ClientWebSocket.WriteJSON(updater)
				if err != nil {
					log.Println("Probably the client is away", client)
				}
			}
		}
	}
}
func (wss *WebsocketServer) SetupServer() {
	http.HandleFunc("/", wss.HomePage)
	http.HandleFunc("/ws", wss.wsEndPoint)
	http.HandleFunc("/logout", wss.handleRoomExit)
	//go wss.handleMessages()
	go wss.handleCustomMessages()
	go wss.roomUpdater()
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
		clientRoomActivity: make(chan string),
		userTicker: time.NewTicker(3 * time.Second),
	}
}