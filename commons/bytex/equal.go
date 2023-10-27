package bytex

func Equal(a, b []byte) bool {
	return ToString(a) == ToString(b)
}
