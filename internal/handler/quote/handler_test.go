package quote

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeQuoteClient struct {
	quote string
	err   error
}

func (f fakeQuoteClient) GetQuote() (string, error) {
	return f.quote, f.err
}

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name               string
		quoteClient        fakeQuoteClient
		expectedStatusCode int
		expectedBodySubstr string
	}{
		{
			name: "Success case",
			quoteClient: fakeQuoteClient{
				quote: "The best quote ever",
				err:   nil,
			},
			expectedStatusCode: http.StatusOK,
			expectedBodySubstr: "The best quote ever",
		},
		{
			name: "Error case",
			quoteClient: fakeQuoteClient{
				quote: "",
				err:   errors.New("failed to get quote"),
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBodySubstr: "couldn't get a quote: failed to get quote",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := New(tc.quoteClient)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			h.Handle(rr, req)
			require.Equal(t, tc.expectedStatusCode, rr.Code)
			require.True(t, strings.Contains(rr.Body.String(), tc.expectedBodySubstr),
				"expected response to contain %q, got %q", tc.expectedBodySubstr, rr.Body.String())
		})
	}
}
