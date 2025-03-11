package test

import (
	"encoding/json"
	"hashcash/internal/client"
	"hashcash/internal/dto"
	"hashcash/internal/handler/word_of_wisdom"
	"hashcash/internal/middleware"
	"hashcash/internal/server"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type wordOfWisdomMock struct{}

func (w wordOfWisdomMock) GetQuote() string {
	return "Some cool quote"
}

type keyProvider struct{}

func (k keyProvider) Get() []byte {
	return []byte("some_cool_key")
}

func getFreePort(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().String()
	parts := strings.Split(addr, ":")
	require.GreaterOrEqual(t, len(parts), 2, "unexpected address format: %s", addr)
	return parts[len(parts)-1]
}

func startTestServer(t *testing.T, port string) *server.Server {
	srv := server.NewServer(port, 100*time.Millisecond)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := srv.Start()
		if err != nil {
			t.Logf("Server stopped with error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	return srv
}

func stopTestServer(srv *server.Server) {
	srv.Stop()
}

func parseServerResponse(respStr string) (dto.ServerResponse, error) {
	var sr dto.ServerResponse
	err := json.Unmarshal([]byte(respStr), &sr)
	return sr, err
}

func ResetCache(c *client.Client) {
	c.CacheMu.Lock()
	defer c.CacheMu.Unlock()
	c.CachedPow = nil
}

func TestIntegration_Success(t *testing.T) {
	port := getFreePort(t)
	srv := startTestServer(t, port)
	defer stopTestServer(srv)

	powMiddleware := middleware.New(keyProvider{}, 5*time.Minute, 5)
	wordHandler := word_of_wisdom.New(wordOfWisdomMock{})
	srv.RegisterHandler("getQuote", powMiddleware.HashCashMiddleware(wordHandler.HandleMessage))

	cli := client.NewClient("127.0.0.1", port, 100*time.Millisecond)
	ResetCache(cli)

	respStr, err := cli.GetQuote()
	require.NoError(t, err)

	var sr dto.ServerResponse
	err = json.Unmarshal([]byte(respStr), &sr)
	require.NoError(t, err)
	require.Equal(t, 0, sr.Code)
	require.NotNil(t, sr.Data)
	require.Equal(t, "Some cool quote", *sr.Data)
}

func TestIntegration_WrongChallenge(t *testing.T) {
	port := getFreePort(t)
	srv := startTestServer(t, port)
	defer stopTestServer(srv)

	powMiddleware := middleware.New(keyProvider{}, 5*time.Minute, 5)
	wordHandler := word_of_wisdom.New(wordOfWisdomMock{})
	srv.RegisterHandler("getQuote", powMiddleware.HashCashMiddleware(wordHandler.HandleMessage))

	cli := client.NewClient("127.0.0.1", port, 500*time.Millisecond)
	ResetCache(cli)

	_, err := cli.GetQuote()
	require.NoError(t, err)

	cli.CacheMu.Lock()
	cli.CachedPow.Nonce = "wrong-nonce"
	cli.CacheMu.Unlock()

	_, err = cli.GetQuote()
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong result of PoW")
}

func TestIntegration_TTLFailure(t *testing.T) {
	port := getFreePort(t)
	srv := startTestServer(t, port)
	defer stopTestServer(srv)

	powMiddleware := middleware.New(keyProvider{}, 5*time.Minute, 5)
	wordHandler := word_of_wisdom.New(wordOfWisdomMock{})
	srv.RegisterHandler("getQuote", powMiddleware.HashCashMiddleware(wordHandler.HandleMessage))

	cli := client.NewClient("127.0.0.1", port, 100*time.Millisecond)
	ResetCache(cli)

	_, err := cli.GetQuote()
	require.NoError(t, err)

	cli.CacheMu.Lock()
	cli.CachedPow.Challenge.Timestamp = time.Now().Add(-time.Second).Unix()
	cli.CacheMu.Unlock()

	_, err = cli.GetQuote()
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong token")
}

func TestIntegration_IPMismatch(t *testing.T) {
	port := getFreePort(t)
	srv := startTestServer(t, port)
	defer stopTestServer(srv)

	powMiddleware := middleware.New(keyProvider{}, 5*time.Minute, 5)
	wordHandler := word_of_wisdom.New(wordOfWisdomMock{})
	srv.RegisterHandler("getQuote", powMiddleware.HashCashMiddleware(wordHandler.HandleMessage))

	cli := client.NewClient("127.0.0.1", port, 500*time.Millisecond)
	ResetCache(cli)

	_, err := cli.GetQuote()
	require.NoError(t, err)

	cli.CacheMu.Lock()
	cli.CachedPow.Token = "invalid-token"
	cli.CacheMu.Unlock()

	_, err = cli.GetQuote()
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong token")
}

func TestIntegration_InvalidFields(t *testing.T) {
	port := getFreePort(t)
	srv := server.NewServer(port, 500*time.Millisecond)
	go func() {
		_ = srv.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer stopTestServer(srv)

	cli := client.NewClient("127.0.0.1", port, 500*time.Millisecond)
	ResetCache(cli)

	cli.CacheMu.Lock()
	cli.CachedPow = &client.CachedPow{
		Challenge: dto.ChallengeDto{
			Salt:       "abc",
			Timestamp:  time.Now().Unix(),
			Difficulty: 1,
		},
		Token:      "dummy-token",
		Nonce:      "0",
		TokenTtlMs: 300 * time.Millisecond,
	}
	cli.CacheMu.Unlock()

	_, err := cli.GetQuote()
	require.Error(t, err)
	require.Contains(t, err.Error(), "undefined method")
}
