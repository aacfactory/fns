package cors

import "github.com/aacfactory/fns/commons/bytex"

func convert(s [][]byte, converter func(string) string) [][]byte {
	out := make([][]byte, 0, len(s))
	for _, i := range s {
		out = append(out, bytex.FromString(converter(bytex.ToString(i))))
	}
	return out
}
