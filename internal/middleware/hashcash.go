package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hashcash/internal/dto"
	"hashcash/internal/server"
	"math/big"
	"strings"
	"time"
)

type hashCashMiddleware struct {
	secretKeyProvider  SecretKeyProvider
	tokenTtl           time.Duration
	challengeDifficult int
}

func New(secretKeyProvider SecretKeyProvider, tokenTtl time.Duration, challengeDifficult int) hashCashMiddleware {
	return hashCashMiddleware{
		secretKeyProvider:  secretKeyProvider,
		tokenTtl:           tokenTtl,
		challengeDifficult: challengeDifficult,
	}
}

func (h hashCashMiddleware) generateChallengeResponse(ip string) ([]byte, error) {
	salt, err := generateRandomSalt(16)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate salt, %v", err)
	}
	challenge := dto.ChallengeDto{
		Salt:       salt,
		Timestamp:  time.Now().Unix(),
		Difficulty: h.challengeDifficult,
	}

	token, err := h.createToken(challenge, ip)
	if err != nil {
		return nil, fmt.Errorf("couldn't create token: %v", err)
	}
	response := dto.ChallengeResponseDto{
		Challenge:  challenge,
		Token:      token,
		TokenTtlMs: h.tokenTtl.Milliseconds(),
	}

	return json.Marshal(response)
}

func (h hashCashMiddleware) createToken(challenge dto.ChallengeDto, ip string) (string, error) {
	challengeBytes, err := json.Marshal(struct {
		Salt       string `json:"salt"`
		Timestamp  int64  `json:"timestamp"`
		Difficulty int    `json:"difficulty"`
		Ip         string `json:"ip"`
	}{
		Salt:       challenge.Salt,
		Timestamp:  challenge.Timestamp,
		Difficulty: challenge.Difficulty,
		Ip:         ip,
	})
	if err != nil {
		return "", fmt.Errorf("couldn't marshal a challenge: %v", err)
	}

	mac := hmac.New(sha256.New, h.secretKeyProvider.Get())
	mac.Write(challengeBytes)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (h hashCashMiddleware) HashCashMiddleware(next server.MessageHandler) server.MessageHandler {
	return func(payload []byte) ([]byte, error) {
		var pow dto.PowPayload
		if err := json.Unmarshal(payload, &pow); err != nil {
			return nil, fmt.Errorf("wrong format payload: %v", err)
		}

		if strings.TrimSpace(pow.Method) == "" {
			return nil, fmt.Errorf("method is empty in payload")
		}

		if len(strings.TrimSpace(pow.Ip)) == 0 {
			return nil, fmt.Errorf("coudn't define an ip")
		}

		//if no token - generate a task
		if strings.TrimSpace(pow.Token) == "" {
			challengeResp, err := h.generateChallengeResponse(pow.Ip)
			if err != nil {
				return nil, fmt.Errorf("couldn't generate a challenge: %v", err)
			}
			return challengeResp, nil
		}

		//if challenge is outdated returns error
		currentTime := time.Now().Unix()
		if currentTime-pow.Challenge.Timestamp > h.tokenTtl.Milliseconds() {
			return nil, fmt.Errorf("challenge is outdated")
		}

		//create a token to compare with client token
		expectedToken, err := h.createToken(dto.ChallengeDto{
			Salt:       pow.Challenge.Salt,
			Timestamp:  pow.Challenge.Timestamp,
			Difficulty: pow.Challenge.Difficulty,
		}, pow.Ip)
		if err != nil {
			return nil, fmt.Errorf("couldn't create a token: %v", err)
		}

		//compare tokens
		if !hmac.Equal([]byte(expectedToken), []byte(pow.Token)) {
			return nil, fmt.Errorf("wrong token")
		}

		//check the answer of PoW
		combined := pow.Challenge.Salt + fmt.Sprintf("%d", pow.Challenge.Timestamp) + pow.Nonce
		h := sha256.New()
		h.Write([]byte(combined))
		hashHex := fmt.Sprintf("%x", h.Sum(nil))
		requiredPrefix := strings.Repeat("0", pow.Challenge.Difficulty)
		if !strings.HasPrefix(hashHex, requiredPrefix) {
			return nil, fmt.Errorf("wrong result of PoW")
		}

		return next(pow.Body)
	}
}

func generateRandomSalt(n int) (string, error) {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	length := len(letters)
	bigint := big.NewInt(int64(length))
	rand, err := rand.Int(rand.Reader, bigint)
	if err != nil {
		return "", err
	}
	for i := range b {
		b[i] = letters[rand.Int64()]
	}
	return string(b), nil
}
