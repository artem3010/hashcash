package quote

type quoteClient interface {
	GetQuote() (string, error)
}
