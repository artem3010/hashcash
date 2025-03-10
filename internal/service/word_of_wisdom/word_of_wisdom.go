package word_of_wisdom

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
)

type service struct {
	quotes []string
}

func New(filePath string) (service, error) {
	quotes := make([]string, 0, 2000)
	file, err := os.Open(filePath)
	if err != nil {
		return service{}, fmt.Errorf("couldn't open a file %s, %v", filePath, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		quotes = append(quotes, line)
	}

	if err := scanner.Err(); err != nil {
		return service{}, fmt.Errorf("couldn't open a file %s, %v", filePath, err)
	}

	return service{
		quotes: quotes,
	}, nil

}

func (s service) GetQuote() string {
	return s.quotes[rand.Intn(len(s.quotes))]
}
