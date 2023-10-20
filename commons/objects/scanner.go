package objects

import (
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
)

const (
	nilJson        = "null"
	emptyJson      = "{}"
	emptyArrayJson = "[]"
)

func NewScanner(data interface{}) Scanner {
	return Scanner{
		data: data,
	}
}

type Scanner struct {
	data interface{}
}

func (scanner Scanner) Exist() (ok bool) {
	if scanner.data == nil {
		return
	}
	switch scanner.data.(type) {
	case []byte:
		p := scanner.data.([]byte)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	case json.RawMessage:
		p := scanner.data.(json.RawMessage)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	case stdjson.RawMessage:
		p := scanner.data.(stdjson.RawMessage)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	default:
		ok = true
		break
	}
	return
}

func (scanner Scanner) Scan(v interface{}) (err error) {
	if scanner.data == nil {
		return
	}
	switch scanner.data.(type) {
	case []byte, json.RawMessage, stdjson.RawMessage:
		var value []byte
		switch scanner.data.(type) {
		case []byte:
			value = scanner.data.([]byte)
			break
		case json.RawMessage:
			value = scanner.data.(json.RawMessage)
			break
		case stdjson.RawMessage:
			value = scanner.data.(stdjson.RawMessage)
			break
		}
		if len(value) == 0 {
			return
		}
		switch v.(type) {
		case *json.RawMessage:
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
		case *stdjson.RawMessage:
			vv := v.(*stdjson.RawMessage)
			*vv = append(*vv, value...)
		case *[]byte:
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
		default:
			if nilJson == bytex.ToString(value) || emptyJson == bytex.ToString(value) || emptyArrayJson == bytex.ToString(value) {
				return
			}
			decodeErr := json.Unmarshal(value, v)
			if decodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(decodeErr)
				return
			}
		}
		return
	default:
		switch v.(type) {
		case *json.RawMessage:
			value, encodeErr := json.Marshal(scanner.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
			break
		case *stdjson.RawMessage:
			value, encodeErr := json.Marshal(scanner.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*stdjson.RawMessage)
			*vv = append(*vv, value...)
			break
		case *[]byte:
			value, encodeErr := json.Marshal(scanner.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
			break
		default:
			cpErr := CopyInterface(v, scanner.data)
			if cpErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(cpErr)
				return
			}
		}
	}
	return
}

func (scanner Scanner) MarshalJSON() (p []byte, err error) {
	if scanner.data == nil {
		p = bytex.FromString(nilJson)
		return
	}
	switch scanner.data.(type) {
	case []byte:
		x := scanner.data.([]byte)
		if json.Validate(x) {
			p = x
		} else {
			p, err = json.Marshal(scanner.data)
		}
	case json.RawMessage:
		p = scanner.data.(json.RawMessage)
	case stdjson.RawMessage:
		p = scanner.data.(stdjson.RawMessage)
	default:
		p, err = json.Marshal(scanner.data)
	}
	return
}
