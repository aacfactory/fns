package cors

import "bytes"

var (
	comma = []byte{','}
)

func parseHeaderList(headerList [][]byte) [][]byte {
	out := headerList
	copied := false
	for i, v := range headerList {
		needsSplit := bytes.IndexByte(v, ',') != -1
		if !copied {
			if needsSplit {
				split := bytes.Split(v, comma)
				out = make([][]byte, i, len(headerList)+len(split)-1)
				copy(out, headerList[:i])
				for _, s := range split {
					out = append(out, bytes.TrimSpace(s))
				}
				copied = true
			}
		} else {
			if needsSplit {
				split := bytes.Split(v, comma)
				for _, s := range split {
					out = append(out, bytes.TrimSpace(s))
				}
			} else {
				out = append(out, v)
			}
		}
	}
	return out
}
