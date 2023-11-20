package sources

import "github.com/aacfactory/gcg"

type FnAnnotationCodeWriter interface {
	Annotation() (annotation string)
	Before() (code gcg.Code, err error)
	After() (code gcg.Code, err error)
}

type FnAnnotationCodeWriters []FnAnnotationCodeWriter

func (writers FnAnnotationCodeWriters) Get(annotation string) (w FnAnnotationCodeWriter, has bool) {
	for _, writer := range writers {
		if writer.Annotation() == annotation {
			w = writer
			has = true
			return
		}
	}
	return
}
