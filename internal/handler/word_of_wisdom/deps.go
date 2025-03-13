package word_of_wisdom

type quoteProvider interface {
	GetQuote() string
}
