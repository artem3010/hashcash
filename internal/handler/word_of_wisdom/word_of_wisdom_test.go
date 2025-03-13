package word_of_wisdom

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeQuoteProvider struct {
	quote string
}

func (f fakeQuoteProvider) GetQuote() string {
	return f.quote
}

func TestHandler_HandleMessage(t *testing.T) {
	tests := []struct {
		name             string
		providedQuote    string
		expectedResponse string
	}{
		{
			name:             "Simple quote",
			providedQuote:    "Hello, world!",
			expectedResponse: "Hello, world!",
		},
		{
			name:             "Empty quote",
			providedQuote:    "",
			expectedResponse: "",
		},
		{
			name:             "Quote with spaces",
			providedQuote:    "   Some wisdom   ",
			expectedResponse: "   Some wisdom   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := fakeQuoteProvider{quote: tt.providedQuote}
			h := New(fp)
			resp, err := h.HandleMessage(nil)
			require.NoError(t, err)
			require.Equal(t, tt.expectedResponse, string(resp))
		})
	}
}
