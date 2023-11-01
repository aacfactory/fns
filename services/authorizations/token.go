package authorizations

type Token []byte

func (token Token) String() string {
	return string(token)
}
