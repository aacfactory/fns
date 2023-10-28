package standard

import (
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"net/http"
)

func HttpTransportHandlerAdaptor(h transports.Handler, maxRequestBody int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		r := &Request{
			maxBodySize: maxRequestBody,
			userValues:  make(objects.Entries, 0, 1),
			request:     request,
		}

		buf := bytebufferpool.Get()
		w := convertHttpResponseWriterToResponseWriter(writer, buf)

		h.Handle(w, r)

		writer.WriteHeader(w.Status())

		bodyLen := buf.Len()
		if bodyLen > 0 {
			body := buf.Bytes()
			n := 0
			for n < bodyLen {
				nn, writeErr := writer.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		bytebufferpool.Put(buf)
	})
}
