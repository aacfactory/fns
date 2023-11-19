package proxy

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"strconv"
)

var (
	managerHandlerPath = append(handlerPathPrefix, []byte("/clusters/manager")...)
)

func FetchEndpointInfos(ctx context.Context, client transports.Client, signature signatures.Signature) (infos services.EndpointInfos, err error) {
	cmd := Command{
		Command: "infos",
		Payload: nil,
	}
	body, sign, bodyErr := encodeCommand(cmd, signature)
	if bodyErr != nil {
		err = errors.Warning("fns: fetch endpoint infos failed").WithCause(bodyErr)
		return
	}
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	header.Set(bytex.FromString(transports.SignatureHeaderName), sign)
	status, _, respBody, doErr := client.Do(ctx, transports.MethodPost, managerHandlerPath, header, body)
	if doErr != nil {
		err = errors.Warning("fns: fetch endpoint infos failed").WithCause(doErr)
		return
	}
	if status == 200 {
		infos = make(services.EndpointInfos, 0, 1)
		err = json.Unmarshal(respBody, &infos)
		if err != nil {
			err = errors.Warning("fns: fetch endpoint infos failed").WithCause(err)
			return
		}
		return
	}
	err = errors.Warning("fns: fetch endpoint infos failed").WithCause(errors.Warning("status is not ok").WithMeta("status", strconv.Itoa(status)))
	return
}

func NewManagerHandler(manager services.EndpointsManager) transports.Handler {
	return &ManagerHandler{
		manager: manager,
	}
}

type ManagerHandler struct {
	manager services.EndpointsManager
}

func (handler *ManagerHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	cmd, cmdErr := ParseCommand(r)
	if cmdErr != nil {
		w.Failed(cmdErr)
		return
	}
	switch cmd.Command {
	case "infos":
		infos := handler.manager.Info()
		w.Succeed(infos)
		break
	default:
		w.Failed(errors.Warning("fns: invalid proxy command"))
	}
}
