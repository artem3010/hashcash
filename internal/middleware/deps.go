package middleware

type SecretKeyProvider interface {
	Get() []byte
}
