package quote

import "net/http"

type handler struct {
	quoteClient quoteClient
}

func New(quoteClient quoteClient) handler {
	return handler{
		quoteClient: quoteClient,
	}
}

func (h handler) Handle(w http.ResponseWriter, _ *http.Request) {
	quote, err := h.quoteClient.GetQuote()
	if err != nil {
		http.Error(w, "couldn't get a quote: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(quote))
}
