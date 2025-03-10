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
)

type MessageHandler func(payload []byte) (response []byte, err error)

type Middleware func(MessageHandler) MessageHandler

type Server struct {
	Addr     string
	listener net.Listener
	done     chan struct{}
	wg       sync.WaitGroup
	handlers map[string]MessageHandler
}

func NewServer(addr string) *Server {
	return &Server{
		Addr:     ":" + addr,
		done:     make(chan struct{}),
		handlers: make(map[string]MessageHandler),
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
	fmt.Println("server started at", s.Addr)

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
				fmt.Println("couldn't set a connection:", err)
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
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("couldn't read data:", err)
			}
			break
		}
		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			continue
		}
		resp := s.dispatchMessage(conn, data)
		if len(resp) > 0 {
			_, err := conn.Write(append(resp, '\n'))
			if err != nil {
				fmt.Println("couldn't send a response:", err)
				break
			}
		}
	}
	conn.Close()
}

func (s *Server) dispatchMessage(conn net.Conn, data []byte) []byte {

	remoteAddr := conn.RemoteAddr().String()
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

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
