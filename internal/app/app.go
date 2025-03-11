package app

import (
	"fmt"
	"hashcash/internal/handler/word_of_wisdom"
	"hashcash/internal/middleware"
	"hashcash/internal/pkg/env"
	"hashcash/internal/server"
	"hashcash/internal/service/secret_key_provider"
	word_of_wisdom_service "hashcash/internal/service/word_of_wisdom"

	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type app struct {
}

func New() *app {
	return &app{}
}

func (a app) Run() error {
	env.LoadEnv()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	port := env.GetEnv("SERVER_PORT", "8080")
	filePath := env.GetEnv("WORD_OF_WISDOM_SOURCE", "word_of_wisdom")
	difficult, err := strconv.Atoi(env.GetEnv("CHALLENGE_DIFFICULT", "5"))
	if err != nil {
		return fmt.Errorf("couldn't parse challenge difficult %v", err)
	}

	tokenTtl, err := time.ParseDuration(env.GetEnv("TOKEN_TTL", "5m"))
	if err != nil {
		return fmt.Errorf("couldn't parse token ttl %v", err)
	}
	connectionDeadline, err := time.ParseDuration(env.GetEnv("CONNECTION_DEADLINE", "100ms"))
	if err != nil {
		return fmt.Errorf("couldn't parse connectionDeadline %v", err)
	}
	tcpServer := server.NewServer(port, connectionDeadline)
	keyProvider := secret_key_provider.New()
	powMiddleware := middleware.New(keyProvider, tokenTtl, difficult)
	wordOfWisdomService, err := word_of_wisdom_service.New(filePath)
	if err != nil {
		return fmt.Errorf("couldn't create a word of wisdom service %v", err)
	}

	wordOfWisdomHandler := word_of_wisdom.New(wordOfWisdomService)
	tcpServer.RegisterHandler("getQuote", powMiddleware.HashCashMiddleware(wordOfWisdomHandler.HandleMessage))

	go func() {
		if err := tcpServer.Start(); err != nil {
			log.Fatal().
				Err(err).
				Msg("couldn't start a server on the port " + port)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	_ = <-sigChan

	tcpServer.Stop()

	return nil
}
