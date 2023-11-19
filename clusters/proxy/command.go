package proxy

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
)

type Command struct {
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

func ParseCommand(r transports.Request) (cmd Command, err error) {
	body, bodyErr := r.Body()
	if bodyErr != nil {
		err = errors.Warning("fns: parse proxy command failed").WithCause(bodyErr)
		return
	}
	err = json.Unmarshal(body, &cmd)
	if err != nil {
		err = errors.Warning("fns: parse proxy command failed").WithCause(err)
		return
	}
	return
}

func encodeCommand(cmd Command, signature signatures.Signature) (body []byte, sign []byte, err error) {
	body, err = json.Marshal(cmd)
	if err != nil {
		err = errors.Warning("fns: encode proxy command failed").WithCause(err)
		return
	}
	sign = signature.Sign(body)
	return
}
