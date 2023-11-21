package writers

import "github.com/aacfactory/gcg"

func contextCode() gcg.Code {
	return gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/context"), "Context")
}
