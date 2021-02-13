package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/uds5501/re-formers-server/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "1337"
		log.Fatal("No port was found, port set to", port)
	}
	fmt.Println("Go WebSockets")
	wss := server.Init()
	wss.SetupServer()
	log.Fatal(http.ListenAndServe(":"+port, nil))
}