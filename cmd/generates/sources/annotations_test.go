package sources_test

import (
	"fmt"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	s := `@title title
@desc >>>

	name1:
	zh: chinese
	en: english
	name2:
	zh: chinese
	en: english

<<<
@barrier
@auth
@permission
@sql:tx name
@cache get set`
	annos, err := sources.ParseAnnotations(s)
	if err != nil {
		fmt.Println(fmt.Sprintf("%+v", err))
		return
	}
	for _, anno := range annos {
		fmt.Println(anno.Name, len(anno.Params), fmt.Sprintf("%+v", anno.Params))
	}
}
