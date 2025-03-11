package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hashcash/internal/dto"
	"hashcash/internal/pkg/ptr"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type MessageHandler func(payload []byte) (response []byte, err error)

type Middleware func(MessageHandler) MessageHandler

type Server struct {
	Addr               string
	listener           net.Listener
	done               chan struct{}
	wg                 sync.WaitGroup
	handlers           map[string]MessageHandler
	connectionDeadline time.Duration
}

func NewServer(addr string, connectionDeadline time.Duration) *Server {
	return &Server{
		Addr:               ":" + addr,
		done:               make(chan struct{}),
		handlers:           make(map[string]MessageHandler),
		connectionDeadline: connectionDeadline,
	}
}

func (s *Server) RegisterHandler(method string, handler MessageHandler) {
	s.handlers[method] = handler
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	log.Info().Msg("server started at " + s.Addr)

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Err(err).Msg("couldn't set a connection")
				continue
			}
		}
		s.wg.Add(1)
		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer s.wg.Done()
	reader := bufio.NewReader(conn)
	//set deadline
	err := conn.SetDeadline(time.Now().Add(s.connectionDeadline))
	if err != nil {
		log.Err(err).Msg("couldn't set a deadline")
		return
	}
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Err(err).Msg("couldn't read data")
			}
			break
		}
		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			continue
		}
		log.Debug().Msg("Received request, processing: " + string(data))
		resp := s.dispatchMessage(conn, data)
		if len(resp) > 0 {
			log.Debug().Msg("Sending response: " + string(resp))
			_, err := conn.Write(append(resp, '\n'))
			if err != nil {
				log.Err(err).Msg("couldn't send a response:")
				break
			}
		}
	}
	conn.Close()
}

func (s *Server) dispatchMessage(conn net.Conn, data []byte) []byte {

	if string(data) == "PING" {
		return wrapResponse(0, []byte("PONG"), nil)
	}

	remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return wrapResponse(1, nil, ptr.Ptr("couldn't get an ip"))
	}
	ip := remoteAddr.IP.String()

	var payload dto.PowPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return wrapResponse(1, nil, ptr.Ptr("wrong format"))
	}

	payload.Ip = ip

	newData, err := json.Marshal(payload)
	if err != nil {
		return wrapResponse(1, nil, ptr.Ptr(fmt.Sprintf("couldn't unmarshal: %v", err)))
	}

	handler, ok := s.handlers[payload.Method]
	if !ok {
		return wrapResponse(1, nil, ptr.Ptr(fmt.Sprintf("undefined method '%s'", payload.Method)))
	}
	resp, err := handler(newData)
	if err != nil {
		return wrapResponse(1, nil, ptr.Ptr("Error: "+err.Error()))
	}
	return wrapResponse(0, resp, nil)
}

func (s *Server) Stop() {
	close(s.done)
	s.listener.Close()
	s.wg.Wait()
}

func wrapResponse(code int, data []byte, errStr *string) []byte {
	d := string(data)
	resp := dto.ServerResponse{
		Code:  code,
		Data:  &d,
		Error: errStr,
	}
	if data == nil {
		resp.Data = nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte(fmt.Sprintf(`{"code":1, "error":"internal error"}`))
	}
	return b
}
