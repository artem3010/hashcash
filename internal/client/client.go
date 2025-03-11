package client

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hashcash/internal/dto"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CachedPow struct {
	Challenge  dto.ChallengeDto
	Token      string
	Nonce      string
	TokenTtlMs time.Duration
}

type Client struct {
	Addr        string
	CachedPow   *CachedPow
	CacheMu     sync.Mutex
	ReadTimeout time.Duration
}

func NewClient(url, port string, pingTimeout time.Duration) *Client {
	return &Client{
		Addr:        url + ":" + port,
		CachedPow:   nil,
		ReadTimeout: pingTimeout,
	}
}

func (c *Client) pingServer(conn net.Conn) error {
	_, err := fmt.Fprintf(conn, "PING\n")
	if err != nil {
		return fmt.Errorf("couldn't send ping: %v", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("couldn't read pong: %v", err)
	}
	resp = strings.TrimSpace(resp)

	var sr dto.ServerResponse
	if err := json.Unmarshal([]byte(resp), &sr); err != nil {
		return fmt.Errorf("couldn't unmarshal pong response: %v", err)
	}
	if sr.Code != 0 || sr.Data == nil || *sr.Data != "PONG" {
		return fmt.Errorf("unexpected pong response: %s", resp)
	}
	return nil
}

func (c *Client) requestChallenge(conn net.Conn) (*dto.ChallengeResponseDto, error) {
	initialPayload := dto.PowPayload{
		Method: "getQuote",
		Token:  "",
		Nonce:  "",
		Body:   json.RawMessage("{}"),
	}
	reqBytes, err := json.Marshal(initialPayload)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal: %v", err)
	}
	_, err = fmt.Fprintf(conn, "%s\n", reqBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't send a request: %v", err)
	}

	reader := bufio.NewReader(conn)

	respStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("couldn't read an response: %v", err)
	}
	respStr = strings.TrimSpace(respStr)

	var serverResp dto.ServerResponse
	if err := json.Unmarshal([]byte(respStr), &serverResp); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal a response: %v", err)
	}

	if serverResp.Code != 0 {
		if serverResp.Error != nil {
			return nil, fmt.Errorf("server returned error: %s", *serverResp.Error)
		}
		return nil, fmt.Errorf("server returned uknown error")
	}

	if serverResp.Data == nil {
		return nil, fmt.Errorf("response is empty")
	}

	var challengeResp dto.ChallengeResponseDto
	if err := json.Unmarshal([]byte(*serverResp.Data), &challengeResp); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal response error: %v", err)
	}

	return &challengeResp, nil
}

func computePoW(challenge dto.ChallengeDto, ttlMs int64) (string, error) {
	salt := challenge.Salt
	timestampStr := strconv.FormatInt(challenge.Timestamp, 10)
	difficulty := challenge.Difficulty
	requiredPrefix := strings.Repeat("0", difficulty)
	var nonce string
	var hashHex string
	startTime := time.Now()
	for i := 0; ; i++ {
		candidate := fmt.Sprintf("%d", i)
		combined := salt + timestampStr + candidate
		hash := sha256.Sum256([]byte(combined))
		hashHex = hex.EncodeToString(hash[:])
		if strings.HasPrefix(hashHex, requiredPrefix) {
			nonce = candidate
			break
		}
		if time.Since(startTime) > time.Duration(ttlMs)*time.Millisecond {
			return "", fmt.Errorf("didn't find an result in the required time")
		}
	}
	return nonce, nil
}

func expiredTtl(timestamp int64, ttl time.Duration) bool {
	challengeTime := time.Unix(timestamp, 0)
	return time.Since(challengeTime) >= ttl
}

func (c *Client) GetQuote() (string, error) {
	conn, err := net.Dial("tcp", c.Addr)

	if err != nil {
		return "", fmt.Errorf("couldn't conncect: %s, %v", c.Addr, err)
	}

	deadline := time.Now().Add(c.ReadTimeout)
	err = conn.SetDeadline(deadline)
	if err != nil {
		return "", fmt.Errorf("couldn't set a deadline: %v", err)
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	if c.CachedPow != nil && !expiredTtl(c.CachedPow.Challenge.Timestamp, c.CachedPow.TokenTtlMs) {
		c.CacheMu.Lock()
		cp := c.CachedPow
		c.CacheMu.Unlock()

		if c.pingServer(conn) != nil {
			conn, err = net.Dial("tcp", c.Addr)
			if err != nil {
				return "", fmt.Errorf("couldn't conncect: %s, %v", c.Addr, err)
			}
		}
		return c.sendFinalRequest(conn, cp)
	}

	if c.pingServer(conn) != nil {
		conn, err = net.Dial("tcp", c.Addr)
		if err != nil {
			return "", fmt.Errorf("couldn't conncect: %s, %v", c.Addr, err)
		}
	}
	chResp, err := c.requestChallenge(conn)
	if err != nil {
		if c.pingServer(conn) != nil {
			conn, err = net.Dial("tcp", c.Addr)
			if err != nil {
				return "", fmt.Errorf("couldn't conncect: %s, %v", c.Addr, err)
			}
		}
		chResp, err = c.requestChallenge(conn)
		if err != nil {
			return "", err
		}
	}
	nonce, err := computePoW(chResp.Challenge, chResp.TokenTtlMs)
	if err != nil {
		return "", err
	}
	newCached := &CachedPow{
		Challenge:  chResp.Challenge,
		Token:      chResp.Token,
		Nonce:      nonce,
		TokenTtlMs: time.Duration(chResp.TokenTtlMs) * time.Millisecond,
	}

	c.CacheMu.Lock()
	if c.CachedPow == nil {
		c.CachedPow = newCached
	}
	cp := c.CachedPow
	c.CacheMu.Unlock()

	if c.pingServer(conn) != nil {
		conn, err = net.Dial("tcp", c.Addr)
		if err != nil {
			return "", fmt.Errorf("couldn't conncect: %s, %v", c.Addr, err)
		}
	}
	retries := 0
	resp, err := c.sendFinalRequest(conn, cp)
	for ; err != nil && retries < 4; retries++ {
		resp, err = c.sendFinalRequest(conn, cp)
	}
	if len(resp) == 0 || err != nil {
		return "", fmt.Errorf("couldn't send a request, %v", err)
	}
	return c.convertResponse(resp)
}

func (c *Client) convertResponse(resp string) (string, error) {
	var serverResp dto.ServerResponse
	if err := json.Unmarshal([]byte(resp), &serverResp); err != nil {
		return "", fmt.Errorf("couldn't unmarshal a response: %v", err)
	}
	if serverResp.Code == 1 {
		c.CacheMu.Lock()
		c.CachedPow = nil
		c.CacheMu.Unlock()
		if serverResp.Error != nil {
			return "", fmt.Errorf("server error: %s", *serverResp.Error)
		}
		return "", fmt.Errorf("server returned error")
	}

	if serverResp.Data == nil {
		return "", fmt.Errorf("emtpy response")
	}
	return resp, nil
}

func (c *Client) sendFinalRequest(conn net.Conn, cp *CachedPow) (string, error) {
	fullPayload := dto.PowPayload{
		Method:    "getQuote",
		Challenge: cp.Challenge,
		Token:     cp.Token,
		Nonce:     cp.Nonce,
		Body:      json.RawMessage("{}"),
	}
	fullReqBytes, err := json.Marshal(fullPayload)
	if err != nil {
		return "", fmt.Errorf("couldn't marshal: %v", err)
	}
	_, err = fmt.Fprintf(conn, "%s\n", fullReqBytes)
	if err != nil {
		return "", fmt.Errorf("couldn't send a request: %v", err)
	}
	reader := bufio.NewReader(conn)
	finalRespStr, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("couldn't read a response: %v", err)
	}
	return c.convertResponse(strings.TrimSpace(finalRespStr))
}
