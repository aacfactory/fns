package fns

import (
	"bytes"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/valyala/bytebufferpool"
	"io"
)

var _json jsoniter.API

func initJsonApi() {
	_json = jsoniter.ConfigCompatibleWithStandardLibrary
}

func JsonAPI() jsoniter.API {
	return _json
}

func JsonValid(data []byte) bool {
	return jsoniter.Valid(data)
}

func JsonValidString(data string) bool {
	return jsoniter.Valid([]byte(data))
}

func JsonEncode(v interface{}) []byte {
	b, err := JsonAPI().Marshal(v)
	if err != nil {
		panic(fmt.Errorf("json encode failed, target is %v, cause is %v", v, err))
	}
	return b
}

func JsonDecode(data []byte, v interface{}) {
	err := JsonAPI().Unmarshal(data, v)
	if err != nil {
		panic(fmt.Errorf("json decode failed, target is %v, cause is %v", string(data), err))
	}
}

func JsonDecodeFromString(data string, v interface{}) {
	err := JsonAPI().UnmarshalFromString(data, v)
	if err != nil {
		panic(fmt.Errorf("json decode from string failed, target is %v, cause is %v", data, err))
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewJsonObjectFromBytes(b []byte) *JsonObject {
	if b[0] != '{' || b[len(b)-1] != '}' {
		panic(fmt.Errorf("new json object from bytes failed, %s is not json object bytes", string(b)))
	}
	return &JsonObject{
		raw: b,
	}
}

func NewJsonObject() *JsonObject {
	return &JsonObject{
		raw: []byte{'{', '}'},
	}
}

type JsonObject struct {
	raw []byte
}

func (object *JsonObject) Raw() (raw []byte) {
	raw = object.raw
	return
}

func (object *JsonObject) Contains(path string) (has bool) {
	has = gjson.GetBytes(object.raw, path).Exists()
	return
}

func (object *JsonObject) Get(path string, v interface{}) (err error) {
	if path == "" {
		err = fmt.Errorf("json object get failed, path is empty")
		return
	}
	if v == nil {
		err = fmt.Errorf("json object get %s failed, value is nil", path)
		return
	}
	r := gjson.GetBytes(object.raw, path)
	if !r.Exists() {
		err = fmt.Errorf("json object get %s failed, not exists", path)
		return
	}
	decodeErr := JsonAPI().UnmarshalFromString(r.Raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("json object get %s failed, decode failed", path)
		return
	}
	return
}

func (object *JsonObject) Put(path string, v interface{}) (err error) {
	if path == "" {
		err = fmt.Errorf("json object set failed, path is empty")
		return
	}
	if v == nil {
		err = fmt.Errorf("json object set %s failed, value is nil", path)
		return
	}
	affected, setErr := sjson.SetBytes(object.raw, path, v)
	if setErr != nil {
		err = fmt.Errorf("json object set %s failed", path)
		return
	}
	object.raw = affected
	return
}

func (object *JsonObject) Rem(path string) (err error) {
	if path == "" {
		err = fmt.Errorf("json object remove failed, path is empty")
		return
	}

	affected, remErr := sjson.DeleteBytes(object.raw, path)
	if remErr != nil {
		err = fmt.Errorf("json object remove %s failed", path)
		return
	}
	object.raw = affected
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewJsonArrayFromBytes(b []byte) *JsonArray {
	if b[0] != '[' || b[len(b)-1] != ']' {
		panic(fmt.Errorf("new json array from bytes failed, %s is not json array bytes", string(b)))
	}
	return &JsonArray{
		raw: b,
	}
}

func NewJsonArray() *JsonArray {
	return &JsonArray{
		raw: []byte{'[', ']'},
	}
}

type JsonArray struct {
	raw []byte
}

func (array *JsonArray) Raw() (raw []byte) {
	raw = array.raw
	return
}

func (array *JsonArray) Add(values ...interface{}) (err error) {
	if values == nil || len(values) == 0 {
		err = fmt.Errorf("json array add failed, values is empty")
		return
	}
	rb := bytes.NewReader(array.raw)
	nb := bytebufferpool.Get()
	_, _ = io.Copy(nb, rb)
	affected := nb.Bytes()
	bytebufferpool.Put(nb)
	var addErr error
	for i, value := range values {
		if value == nil {
			continue
		}
		affected, addErr = sjson.SetBytes(affected, "-1", value)
		if addErr != nil {
			err = fmt.Errorf("json array add %d failed", i)
			return
		}
	}
	array.raw = affected
	return
}

func (array *JsonArray) Rem(i int) (err error) {
	if i < 0 {
		err = fmt.Errorf("json array remove failed, index is less than 0")
		return
	}
	affected, remErr := sjson.DeleteBytes(array.raw, fmt.Sprintf("%d", i))
	if remErr != nil {
		err = fmt.Errorf("json array remove %d failed", i)
		return
	}
	array.raw = affected
	return
}

func (array *JsonArray) Len() (size int) {
	size = len(gjson.ParseBytes(array.raw).Array())
	return
}

func (array *JsonArray) Get(i int, v interface{}) (err error) {
	if i < 0 || i >= array.Len() {
		err = fmt.Errorf("json array get failed, index is less than 0 or greater than len")
		return
	}
	if v == nil {
		err = fmt.Errorf("json array get %d failed, value is nil", i)
		return
	}
	raw := gjson.ParseBytes(array.raw).Array()[i].Raw
	decodeErr := JsonAPI().UnmarshalFromString(raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("json array get %d failed, decode failed", i)
		return
	}
	return
}
