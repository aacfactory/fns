package sources_test

import (
	"fmt"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	s := `@title title
@desc >>> 
000
desc1
desc2
desc3
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
