package dto

import "encoding/json"

type ChallengeDto struct {
	Salt       string `json:"salt"`
	Timestamp  int64  `json:"timestamp"`
	Difficulty int    `json:"difficulty"`
}

type ChallengeResponseDto struct {
	Challenge  ChallengeDto `json:"challenge"`
	Token      string       `json:"token"`
	TokenTtlMs int64        `json:"ttl"`
}

type PowPayload struct {
	Method    string          `json:"method"`
	Challenge ChallengeDto    `json:"challenge"`
	Token     string          `json:"token"`
	Nonce     string          `json:"nonce"`
	Body      json.RawMessage `json:"body"`
	Ip        string
}

type ServerResponse struct {
	Code  int     `json:"code"`
	Data  *string `json:"data"`
	Error *string `json:"error"`
}
