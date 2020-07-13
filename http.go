package trans

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/shubinmi/util/errs"
)

type HTTPMdl func(next http.HandlerFunc) http.HandlerFunc

type HTTPOpt func(*httpOpt)

type HTTPControllerServe func(opts ...HTTPOpt) http.HandlerFunc

type bodySerializer func(Result, error) []byte

type httpOpt struct {
	opt
	decorators           []HTTPMdl
	paramsExec           ParamsExecutor
	bodySerializeBuilder func(r *http.Request) bodySerializer
}

func WithHTTPDecorators(decorators ...HTTPMdl) HTTPOpt {
	return func(opt *httpOpt) {
		opt.decorators = append(opt.decorators, decorators...)
	}
}
func WithHTTPBodySrlzBuilder(builder func(r *http.Request) bodySerializer) HTTPOpt {
	return func(opt *httpOpt) {
		opt.bodySerializeBuilder = builder
	}
}
func WithParamsExec(f ParamsExecutor) HTTPOpt {
	return func(opt *httpOpt) {
		opt.paramsExec = f
	}
}
func WithControllerEndpoint(f ControllerEndpoint) HTTPOpt {
	return func(opt *httpOpt) {
		opt.opt.ctrlHandler = f
	}
}

func HTTP(opts ...HTTPOpt) HTTPControllerServe {
	h := &httpOpt{
		decorators:           []HTTPMdl{},
		paramsExec:           httpParamsExec,
		bodySerializeBuilder: httpBodySerializeBuilder(),
	}
	for _, o := range opts {
		o(h)
	}
	return h.handler
}
func (h *httpOpt) handler(opts ...HTTPOpt) http.HandlerFunc {
	nh := *h
	nh.decorators = append([]HTTPMdl{validateDecorator(&nh)}, h.decorators...)
	for _, o := range opts {
		o(&nh)
	}
	if !nh.valid() {
		log.Println("be AWARE that handler is invalid")
	}
	ch := func(writer http.ResponseWriter, request *http.Request) {
		d, e := nh.ctrlHandler(request.Context(), nh.paramsExec(request))
		code, e := httpCode(e)
		writer.WriteHeader(code)
		if _, e = writer.Write(nh.bodySerializeBuilder(request)(d, e)); e != nil {
			log.Printf("write response: %s; err=%s", d, e)
		}
	}
	index := len(nh.decorators) - 1
	for i := range nh.decorators {
		ch = nh.decorators[index-i](ch)
	}
	return ch
}
func (h *httpOpt) valid() bool {
	return !(h.bodySerializeBuilder == nil || h.decorators == nil || h.paramsExec == nil || h.ctrlHandler == nil)
}

func httpCode(e error) (int, error) {
	code := http.StatusOK
	if e != nil {
		switch errs.ExecCode(e) {
		case errs.AccessDeny:
			code = http.StatusUnauthorized
		case errs.Internal:
			code = http.StatusInternalServerError
		case errs.NotFound:
			code = http.StatusNotFound
		case errs.Timeout:
			code = http.StatusRequestTimeout
		default:
			code = http.StatusBadRequest
		}
		e = errs.CutCode(e)
	}
	return code, e
}

func validateDecorator(h *httpOpt) HTTPMdl {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			if !h.valid() {
				_, _ = writer.Write([]byte("try to invoke invalid handler"))
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			next(writer, request)
		}
	}
}

func httpBodySerializeBuilder() func(*http.Request) bodySerializer {
	return func(r *http.Request) bodySerializer {
		return func(res Result, err error) []byte {
			e := ""
			if err != nil {
				e = err.Error()
			}
			var data interface{}
			var meta interface{}
			type metaInfo interface {
				Meta() interface{}
			}
			if res != nil {
				data = res.Dto()
				if rm, ok := res.(metaInfo); ok {
					meta = rm.Meta()
				}
			}
			response := struct {
				Data  interface{}
				Meta  interface{}
				Error string
			}{
				Data:  data,
				Error: e,
				Meta:  meta,
			}
			b, err := json.Marshal(response)
			if err != nil {
				return []byte(fmt.Sprintf(`{"Data":"","Error":"%s"}`, e))
			}
			return b
		}
	}
}
