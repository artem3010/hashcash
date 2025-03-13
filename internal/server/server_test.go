package server

import (
	"encoding/json"
	"fmt"
	"hashcash/internal/dto"

	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeConn struct {
	remoteAddr net.Addr
}

func (f fakeConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (f fakeConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (f fakeConn) Close() error                       { return nil }
func (f fakeConn) LocalAddr() net.Addr                { return nil }
func (f fakeConn) RemoteAddr() net.Addr               { return f.remoteAddr }
func (f fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (f fakeConn) SetWriteDeadline(t time.Time) error { return nil }

func fakeTCPAddr(ip string, port int) *net.TCPAddr {
	return &net.TCPAddr{IP: net.ParseIP(ip), Port: port}
}

type customHandler struct {
	calledWith dto.PowPayload
	err        error
	mu         sync.Mutex
}

func (h *customHandler) HandleMessage(payload []byte) ([]byte, error) {
	var p dto.PowPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.calledWith = p
	h.mu.Unlock()
	if h.err != nil {
		return nil, h.err
	}
	return []byte("handler response"), nil
}

func TestDispatchMessage(t *testing.T) {
	srv := NewServer("12345", 100*time.Millisecond)
	fConn := fakeConn{remoteAddr: fakeTCPAddr("127.0.0.1", 54321)}

	tests := []struct {
		name            string
		preparePayload  func() []byte
		registerHandler bool
		handler         MessageHandler
		expectedCode    int
		expectedError   string
		expectedData    string
	}{
		{
			name: "Invalid JSON",
			preparePayload: func() []byte {
				return []byte("not a json")
			},
			expectedCode:  1,
			expectedError: "wrong format",
		},
		{
			name: "Undefined method",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "nonexistent",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: 1},
					Token:     "any",
					Nonce:     "0",
					Body:      json.RawMessage("\"test body\""),
					Ip:        "",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedCode:  1,
			expectedError: "undefined method 'nonexistent'",
		},
		{
			name: "Handler returns error",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "testMethod",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: 1},
					Token:     "valid-token",
					Nonce:     "0",
					Body:      json.RawMessage("\"test body\""),
					Ip:        "",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			registerHandler: true,
			handler: MessageHandler(func(payload []byte) ([]byte, error) {
				return nil, fmt.Errorf("handler error")
			}),
			expectedCode:  1,
			expectedError: "Error: handler error",
		},
		{
			name: "Success case with custom handler",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "testMethod",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: 1},
					Token:     "valid-token",
					Nonce:     "0",
					Body:      json.RawMessage("\"test body\""),
					Ip:        "",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			registerHandler: true,
			handler:         (&customHandler{}).HandleMessage,
			expectedCode:    0,
			expectedData:    "handler response",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.preparePayload()
			if tc.registerHandler {
				var temp dto.PowPayload
				err := json.Unmarshal(input, &temp)
				require.NoError(t, err)
				srv.handlers[temp.Method] = tc.handler
			}

			result := srv.dispatchMessage(fConn, input)

			var resp dto.ServerResponse
			err := json.Unmarshal(result, &resp)
			require.NoError(t, err)
			require.Equal(t, tc.expectedCode, resp.Code)
			if tc.expectedError != "" {
				require.NotNil(t, resp.Error)
				require.Contains(t, *resp.Error, tc.expectedError)
			}
			if tc.expectedData != "" {
				require.NotNil(t, resp.Data)
				require.Equal(t, tc.expectedData, *resp.Data)
			}
			if tc.registerHandler {
				var temp dto.PowPayload
				_ = json.Unmarshal(input, &temp)
				delete(srv.handlers, temp.Method)
			}
		})
	}
}
