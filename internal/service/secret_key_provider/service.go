package secret_key_provider

import "hashcash/internal/pkg/env"

type service struct {
	k []byte
}

func New() service {
	k := env.GetEnv("TOKEN_KEY", "")
	return service{
		k: []byte(k),
	}
}

func (s service) Get() []byte {
	return s.k
}
