package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hashcash/internal/dto"
	"strconv"

	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func dummyNext(body []byte) ([]byte, error) {
	var s string
	err := json.Unmarshal(body, &s)
	if err != nil {
		s = string(body)
	}
	return []byte("next response: " + s), nil
}

type fakeSecretKeyProvider struct {
	secret []byte
}

func (f fakeSecretKeyProvider) Get() []byte {
	return f.secret
}

var (
	testSecret        = []byte("test-secret")
	testTokenTtl      = 300 * time.Millisecond
	testChallengeDiff = 1
)

func newTestMiddleware() hashCashMiddleware {
	return New(fakeSecretKeyProvider{secret: testSecret}, testTokenTtl, testChallengeDiff)
}

func TestHashCashMiddlewareTable(t *testing.T) {
	hm := newTestMiddleware()
	mwHandler := hm.HashCashMiddleware(dummyNext)

	tests := []struct {
		name           string
		preparePayload func() []byte
		expectedErr    string
		expectedOutput string
	}{
		{
			name: "Invalid JSON",
			preparePayload: func() []byte {
				return []byte("not a json")
			},
			expectedErr: "wrong format payload",
		},
		{
			name: "Empty method",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "any",
					Nonce:     "0",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedErr: "method is empty in payload",
		},
		{
			name: "Empty IP",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "any",
					Nonce:     "0",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedErr: "coudn't define an ip",
		},
		{
			name: "Token empty - generate challenge",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "",
					Nonce:     "",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedOutput: "challenge",
		},
		{
			name: "Challenge outdated",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Add(-1000 * time.Second).Unix(), Difficulty: testChallengeDiff},
					Token:     "",
					Nonce:     "0",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				tok, err := hm.createToken(p.Challenge, p.Ip)
				require.NoError(t, err)
				p.Token = tok
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedErr: "challenge is outdated",
		},
		{
			name: "Wrong token",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "invalid-token",
					Nonce:     "0",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedErr: "wrong token",
		},
		{
			name: "Wrong PoW result",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "",
					Nonce:     "9999",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				tok, err := hm.createToken(p.Challenge, p.Ip)
				require.NoError(t, err)
				p.Token = tok
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedErr: "wrong result of PoW",
		},
		{
			name: "Success case",
			preparePayload: func() []byte {
				p := dto.PowPayload{
					Method:    "getQuote",
					Challenge: dto.ChallengeDto{Salt: "abc", Timestamp: time.Now().Unix(), Difficulty: testChallengeDiff},
					Token:     "",
					Nonce:     "",
					Body:      json.RawMessage(`"test body"`),
					Ip:        "127.0.0.1",
				}
				nonce, err := computePoW(p.Challenge, hm.tokenTtl.Milliseconds())
				require.NoError(t, err)
				p.Nonce = nonce
				tok, err := hm.createToken(p.Challenge, p.Ip)
				require.NoError(t, err)
				p.Token = tok
				b, err := json.Marshal(p)
				require.NoError(t, err)
				return b
			},
			expectedOutput: "next response: test body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.preparePayload()
			output, err := mwHandler(input)
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				var incoming dto.PowPayload
				err = json.Unmarshal(input, &incoming)
				require.NoError(t, err)
				if strings.TrimSpace(incoming.Token) == "" {
					var cr dto.ChallengeResponseDto
					err = json.Unmarshal(output, &cr)
					require.NoError(t, err)
					require.NotEmpty(t, cr.Token)
					require.Equal(t, hm.tokenTtl.Milliseconds(), cr.TokenTtlMs)
					return
				}
				require.Equal(t, tc.expectedOutput, string(output))
			}
		})
	}
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
