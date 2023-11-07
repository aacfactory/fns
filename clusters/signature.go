package clusters

import "github.com/aacfactory/fns/commons/signatures"

func NewSignature(secret string) signatures.Signature {
	if secret == "" {
		secret = "FNS+-"
	}
	return signatures.HMAC([]byte(secret))
}
