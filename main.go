package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/uds5501/re-formers-server/server"
)

func main() {
	fmt.Println("Go WebSockets")
	wss := server.Init()
	wss.SetupServer()
	log.Fatal(http.ListenAndServe(":1337", nil))
}