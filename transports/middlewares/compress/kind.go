package compress

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"sort"
	"strconv"
)

const (
	AnyName     = "*"
	DefaultName = "compress"
	GzipName    = "gzip"
	DeflateName = "deflate"
	BrotliName  = "br"
)

const (
	No Kind = iota
	Any
	Default
	Gzip
	Deflate
	Brotli
)

type Kind int

func (kind Kind) String() string {
	switch kind {
	case Default:
		return DefaultName
	case Deflate:
		return DeflateName
	case Gzip:
		return GzipName
	case Brotli:
		return BrotliName
	default:
		return ""
	}
}

type KindWeight struct {
	kind   Kind
	weight float64
}

type KindWeights []KindWeight

func (k KindWeights) Len() int {
	return len(k)
}

func (k KindWeights) Less(i, j int) bool {
	return k[i].weight < k[j].weight
}

func (k KindWeights) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

var (
	comma = []byte{','}
)

func getKind(r transports.Request) Kind {
	accepts := r.Header().Get(bytex.FromString(transports.AcceptEncodingHeaderName))
	if len(accepts) == 0 {
		return No
	}
	items := bytes.Split(accepts, comma)
	kinds := make(KindWeights, 0, len(items))
	weighted := 0
	for _, item := range items {
		kind := No
		weight := 0.0
		idx := bytes.IndexByte(item, ';')
		if idx > 0 {
			weightBytes := bytes.TrimSpace(item[idx+1:])
			if len(weightBytes) > 0 {
				f, parseErr := strconv.ParseFloat(bytex.ToString(weightBytes), 64)
				if parseErr != nil {
					continue
				}
				weight = f
				weighted++
			}
			item = item[0:idx]
		}
		item = bytes.TrimSpace(item)
		switch string(item) {
		case DefaultName:
			kind = Default
			break
		case GzipName:
			kind = Gzip
			break
		case DeflateName:
			kind = Deflate
			break
		case BrotliName:
			kind = Brotli
			break
		case AnyName:
			kind = Any
			break
		default:
			kind = Default
			break
		}
		if kind == No {
			continue
		}
		kinds = append(kinds, KindWeight{
			kind:   kind,
			weight: weight,
		})
	}
	if len(kinds) == 0 {
		return No
	}
	if weighted == 0 {
		return kinds[0].kind
	}
	sort.Sort(kinds)
	return kinds[kinds.Len()-1].kind
}
