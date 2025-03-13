package word_of_wisdom

type handler struct {
	quoteProvider quoteProvider
}

func New(quoteProvider quoteProvider) *handler {
	return &handler{
		quoteProvider: quoteProvider,
	}
}

func (h handler) HandleMessage(_ []byte) (response []byte, err error) {
	return []byte(h.quoteProvider.GetQuote()), nil
}
