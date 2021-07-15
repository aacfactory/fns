package fns

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
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
