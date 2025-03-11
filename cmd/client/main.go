package main

import (
	"hashcash/internal/client"
	"hashcash/internal/handler/quote"
	"hashcash/internal/pkg/env"
	"log"
	"net/http"
	"time"
)

func main() {
	env.LoadEnv()
	serverUrl := env.GetEnv("POW_SERVER_HOST", "")
	serverPort := env.GetEnv("POW_SERVER_PORT", "8080")
	clientPort := env.GetEnv("POW_CLIENT_PORT", "8081")
	pingTimeout, err := time.ParseDuration(env.GetEnv("PING_TIMEOUT", "100ms"))
	if err != nil {
		log.Fatalf("couldn't parse pingTimeout %v", err)
	}
	c := client.NewClient(serverUrl, serverPort, pingTimeout)

	handler := quote.New(c)

	http.HandleFunc("/getQuote", handler.Handle)

	if err := http.ListenAndServe(":"+clientPort, nil); err != nil {
		log.Fatalf("Couldn't start a server: %v", err)
	}
}
