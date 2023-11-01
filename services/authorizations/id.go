package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"strconv"
)

type Id []byte

func (id Id) Int() int64 {
	if len(id) == 0 {
		return 0
	}
	v, err := strconv.ParseInt(id.String(), 10, 64)
	if err != nil {
		panic(errors.Warning("fns: get int value from id failed").WithCause(err).WithMeta("id", id.String()))
		return 0
	}
	return v
}

func (id Id) String() string {
	return bytex.ToString(id)
}

func (id Id) Exist() (ok bool) {
	ok = len(id) > 0
	return
}

func StringId(id []byte) Id {
	return id
}

func IntId(id int64) Id {
	return bytex.FromString(strconv.FormatInt(id, 10))
}
