package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	resetTicker *time.Ticker
	updateActivity chan string

	currFormId int
	formArray []config.FormElement
	formMutex sync.Mutex
	formChannelRequest chan config.FormUpdateElement
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
		//log.Println("Message to distribute: ", newMessage)
		for client := range wss.clients {
			//fmt.Println("SENDING MESSAGE TO ", client)
			err := client.Send(1, []byte(newMessage))
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
			wss.Util.UnlockForm(preListen.EntryToken)
		}
		fmt.Println(e)
	}()
	for {
		var newElement config.ClientRequest
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("Client Disconnected: ",err, preListen.EntryToken)
			//delete(wss.clients, ws)
			break
		}
		err = json.Unmarshal(p, &newElement)
		newElement.WebSocket = ws
		err2, retrievedClient := wss.HandleClientMessage(newElement)
		log.Println("Receieved Client: ", retrievedClient.Colour, retrievedClient.Username, newElement.MessageType)
		preListen = retrievedClient
		if err2 != nil {
			if retrievedClient != nil {
				wss.chuckClient(retrievedClient)
			}
			break
		}
	}
}

func (wss *WebsocketServer) HandleClientMessage(clientData config.ClientRequest) (error, *config.ClientObject){
	//log.Println("client data: ", clientData)
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
			err := clientObj.SendJSON(msg)
			//updater := config.PeriodicUpdater{ClientData: []*config.ClientObject{}, FormData: wss.formArray, MessageType: "history"}
			//err = clientObj.SendJSON(updater)
			if err != nil {
				return err, clientObj
			} else {
				// gonna think about this i guess
				wss.clientRoomActivity <- fmt.Sprintf(`{"MessageType": "user-joined", "userName": "%s", "userColor": "%s"}`, clientObj.Username, clientObj.Colour)
				wss.updateActivity <- "updateUsers"
				wss.updateActivity <- "updateForm"
				return nil, clientObj
			}
		} else {
			// new web socket is in non pre defined token mapping
			//log.Println("we could not find you")
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
						JoinedAt: time.Now(),
					}
					// map entryToken to client object
					wss.clientTokenMap[entryToken] = clientObject
					// map clientObject to a boolean true for easy broadcast
					wss.clients[clientObject] = true
					msg := wss.Util.CreateMessage("welcome", clientObject)
					err := clientObject.SendJSON(msg)
					//updater := config.PeriodicUpdater{ClientData: []*config.ClientObject{}, FormData: wss.formArray, MessageType: "history"}
					//err = clientObj.SendJSON(updater)
					if err != nil {
						return err, clientObject
					} else {
						wss.clientRoomActivity <- fmt.Sprintf(`{"MessageType": "user-joined", "userName": "%s", "userColor": "%s"}`, clientObject.Username, clientObject.Colour)
						wss.updateActivity <- "updateUsers"
						wss.updateActivity <- "updateForm"
						return nil, clientObject
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
	} else if clientData.MessageType == "add element" {
		clientObj, _ := wss.clientTokenMap[clientData.EntryToken]
		formUpdateRequest := config.FormUpdateElement{
			Requester: clientObj,
			Id: -1,
			Action: "add",
			Question: clientData.Question,
			Title: clientData.Title,
		}
		wss.formChannelRequest <- formUpdateRequest
		return nil, clientObj
	} else if clientData.MessageType == "edit" {
		clientObj, _ := wss.clientTokenMap[clientData.EntryToken]
		log.Println("CLIENT DATA:", clientData)
		formUpdateRequest := config.FormUpdateElement{
			Requester: clientObj,
			Id: clientData.FormId,
			Action: "edit",
			Question: clientData.Question,
			Title: clientData.Title,
		}
		wss.formChannelRequest <- formUpdateRequest
		return nil, clientObj
	} else if clientData.MessageType == "delete element" {
		log.Println("In delete element handler", clientData)

		clientObj, _ := wss.clientTokenMap[clientData.EntryToken]
		formUpdateRequest := config.FormUpdateElement{
			Requester: clientObj,
			Id: clientData.FormId,
			Action: "delete",
		}
		wss.formChannelRequest <- formUpdateRequest
		return nil, clientObj
	}
	return nil, nil
}

func (wss *WebsocketServer) addForm (reqObj config.FormUpdateElement) {
	wss.formMutex.Lock()
	defer wss.formMutex.Unlock()
	newObj := config.FormElement{
		Id: wss.currFormId,
		Question: reqObj.Question,
		Title: reqObj.Title,
		CreatedAt: time.Now(),
		Versions: []config.FormVersionControl{},
		IsDeleted: false,
	}
	newObj.Versions = append(newObj.Versions, config.FormVersionControl{
		EditedAt: time.Now(),
		ActionPerformed: "create",
		EditedBy: reqObj.Requester,
		Question: reqObj.Question,
		Title: reqObj.Title,
	})
	wss.formArray = append(wss.formArray, newObj)
	wss.currFormId++
}

func (wss *WebsocketServer) editForm (reqObj config.FormUpdateElement) {
	// we don't need to lock the entire form to edit a particular element
	// use form element lock instead
	formObj := &wss.formArray[reqObj.Id]
	formObj.FormElementLock.Lock()
	defer func() {
		wss.Util.UnlockForm(reqObj.Requester.EntryToken)
		formObj.FormElementLock.Unlock()
	}()
	formObj.Question = reqObj.Question
	formObj.Title = reqObj.Title
	formObj.Versions = append(formObj.Versions, config.FormVersionControl{
		EditedAt: time.Now(),
		ActionPerformed: "edit",
		EditedBy: reqObj.Requester,
		Question: reqObj.Question,
		Title: reqObj.Title,
	})

}

func (wss *WebsocketServer) deleteForm (reqObj config.FormUpdateElement) {
	formObj := &wss.formArray[reqObj.Id]
	clientObj := reqObj.Requester
	if formObj.IsDeleted {
		// make sure you return a notification that it's already deleted
		clientObj.Send(1, []byte(fmt.Sprintf(`{"MessageType": "already-deleted"}`)))
		return
	} else if wss.Util.IsLocked(reqObj.Id) {
		// give the fact that the current object cannot be deleted because it's being edited
		clientObj.Send(1, []byte(fmt.Sprintf(`{"MessageType": "current-locked"}`)))
		return
	}
	formObj.FormElementLock.Lock()
	defer func() {
		formObj.FormElementLock.Unlock()
	}()
	formObj.IsDeleted = true
	clientObj.Send(1, []byte(fmt.Sprintf(`{"MessageType": "delete-confirmed"}`)))
	wss.updateActivity <- "updateForm"
}

func (wss *WebsocketServer) formRequestHandler() {
	for {
		req := <- wss.formChannelRequest
		log.Println("Form request recieved: ", req)
		if req.Action == "add" {
			wss.addForm(req)
			wss.updateActivity <- "updateForm"
		} else if req.Action == "edit" {
			wss.editForm(req)
			wss.updateActivity <- "updateForm"
		} else if req.Action == "delete" {
			wss.deleteForm(req)
		}
	}
}

func (wss *WebsocketServer) handleRoomExit(w http.ResponseWriter, r *http.Request) {
	log.Println("in logout sector")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

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
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, "logged-out")))
		wss.clientRoomActivity <- fmt.Sprintf(`{"MessageType": "user-logout", "userName": "%s", "userColor": "%s"}`, clientObj.Username, clientObj.Colour)
		wss.updateActivity <- "updateUsers"
	}
}

func (wss *WebsocketServer) handleLockAssignment (w http.ResponseWriter, r *http.Request) {
	log.Println("In Lock assignment")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	decoder := json.NewDecoder(r.Body)
	var cr config.ClientRequest
	err := decoder.Decode(&cr)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, "someError")))
	}
	clientObj, found := wss.clientTokenMap[cr.EntryToken]
	if found {
		hookAssigned := wss.Util.AssignLock(clientObj.EntryToken, cr.FormId, wss.formArray[cr.FormId])
		if hookAssigned {
			log.Println("Hook assigned to ", clientObj.Colour, clientObj.Username)
			w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, "assigned")))
		} else {
			log.Println("Hook declined to ", clientObj.Colour, clientObj.Username)
			w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, "declined")))
		}
	}
}

func (wss *WebsocketServer) roomUpdater() {
	for {
		msg := <- wss.updateActivity
		//log.Println("Processing update message", msg)
		if msg == "updateUsers" {
			updater := config.PeriodicUpdater{ClientData: []*config.ClientObject{}, FormData: []config.FormElement{}, MessageType: "updater"}
			for clObj := range wss.clients {
				updater.ClientData = append(updater.ClientData, clObj)
			}
			for client := range wss.clients {
				err := client.SendJSON(updater)
				if err != nil {
					log.Println("Couldn't send update user list to ", client)
				}
			}
		} else if msg == "updateForm" {
			updater := config.PeriodicUpdater{ClientData: []*config.ClientObject{}, FormData: wss.formArray, MessageType: "formUpdater"}
			for client := range wss.clients {
				err := client.SendJSON(updater)
				if err != nil {
					log.Println("Couldn't send updated forms to ", client)
				}
			}
		}
	}
}

func (wss *WebsocketServer) pruneClients() {
	for {
		select {
			case t := <- wss.userTicker.C:
				//log.Println("Checking for prunable users at", t.Format("2006-01-02 15:04:05"))
				for client := range wss.clients {
					if t.Sub(client.JoinedAt) >= 30*time.Minute {
						log.Println("----Disconnecting ", client.Colour, client.Username, "-----")
						// if the client has been logged in for 30 minutes, throw him out
						msg := []byte(fmt.Sprintf(`{"MessageType": "disconnect"}`))
						client.Send(1, msg)
						client.ClientWebSocket.Close()
						wss.Util.UnlockForm(client.EntryToken)
						wss.chuckClient(client)
					}
				}
				wss.updateActivity <- "updateUsers"
				wss.updateActivity <- "updateForm"
		}
	}
}

func (wss *WebsocketServer) handleUnlockAssignment (w http.ResponseWriter, r *http.Request) {
	log.Println("In Client Unlock")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	decoder := json.NewDecoder(r.Body)
	var cr config.ClientRequest
	err := decoder.Decode(&cr)
	if err != nil {
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, "someError")))
	}
	wss.Util.UnlockForm(cr.EntryToken)
}

func (wss *WebsocketServer) initiateReset() {
	for {
		select {
		case t := <- wss.resetTicker.C:
			//log.Println("Checking for prunable users at", t.Format("2006-01-02 15:04:05"))
			for client := range wss.clients {
				if t.Sub(client.JoinedAt) >= 30*time.Minute {
					log.Println("----Disconnecting ", client.Colour, client.Username, "-----")
					// if the client has been logged in for 30 minutes, throw him out
					msg := []byte(fmt.Sprintf(`{"MessageType": "disconnect"}`))
					client.Send(1, msg)
					client.ClientWebSocket.Close()
					wss.Util.UnlockForm(client.EntryToken)
					wss.chuckClient(client)

				}
				wss.formArray = []config.FormElement{}
				wss.clients = make(map[*config.ClientObject]bool)
				wss.clientTokenMap = make(map[string]*config.ClientObject)
				wss.currFormId = 0
				wss.clientRoomActivity = make(chan string)
				wss.updateActivity = make(chan string)
				wss.formChannelRequest = make(chan config.FormUpdateElement)
				wss.Util.Reset()
			}
			wss.updateActivity <- "updateUsers"
			wss.updateActivity <- "updateForm"
		}
	}
}

func (wss *WebsocketServer) SetupServer() {
	http.HandleFunc("/", wss.HomePage)
	http.HandleFunc("/ws", wss.wsEndPoint)
	http.HandleFunc("/logout", wss.handleRoomExit)
	http.HandleFunc("/lock", wss.handleLockAssignment)
	http.HandleFunc("/unlock", wss.handleUnlockAssignment)

	//go wss.handleMessages()
	go wss.handleCustomMessages()
	go wss.roomUpdater()
	go wss.pruneClients()
	for i:=3; i>0; i-- {
		// spawn 3 threads to handle form requests
		go wss.formRequestHandler()
	}
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
		userTicker: time.NewTicker(10 * time.Second),
		resetTicker: time.NewTicker(6 * time.Hour),
		updateActivity: make(chan string),

		currFormId: 0,
		formArray: []config.FormElement{},
		formChannelRequest: make(chan config.FormUpdateElement),
	}
}