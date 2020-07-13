package trans

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"

	gmux "github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/shubinmi/util/errs"
)

type (
	ParamsDtoConsumer func(v interface{}) error
	ParamsExecutor    func(r *http.Request) *Params
	ParamsOpt         func(map[string][]string)
	Params            struct {
		values map[string][]string
		file   func(key string) (multipart.File, *multipart.FileHeader, error)
		sate   struct{ err error }
		dto    interface{}
	}
)

func With(key string, val interface{}) ParamsOpt {
	return func(m map[string][]string) {
		if val == nil {
			return
		}
		m[key] = []string{fmt.Sprint(val)}
	}
}

func NewParams(opts ...ParamsOpt) *Params {
	vs := make(map[string][]string)
	for _, opt := range opts {
		opt(vs)
	}
	return &Params{values: vs}
}

func (p *Params) Enrich(key string, val []string) error {
	if _, ok := p.values[key]; ok {
		return fmt.Errorf("can't enrich params with key '%s'. key already exist", key)
	}
	p.values[key] = val
	return nil
}
func (p *Params) SetDto(val interface{}) *Params {
	p.dto = val
	return p
}
func (p *Params) ScanDto(consume ParamsDtoConsumer) error {
	return consume(p.dto)
}
func (p *Params) ScanStrReq(key string, in *string) *Params {
	v, ok := p.values[key]
	if !ok || v == nil || len(v) == 0 {
		p.sate.err = errs.Merge(p.sate.err, fmt.Errorf("param '%s' is required", key))
		return p
	}
	*in = v[0]
	return p
}
func (p *Params) ScanStr(key string, in *string) *Params {
	v, ok := p.values[key]
	if !ok || v == nil || len(v) == 0 {
		*in = ""
		return p
	}
	*in = v[0]
	return p
}
func (p *Params) ScanStrsReq(key string, in *[]string) *Params {
	v, ok := p.values[key]
	if !ok || v == nil || len(v) == 0 {
		p.sate.err = errs.Merge(p.sate.err, fmt.Errorf("param '%s' is required", key))
		return p
	}
	*in = v
	return p
}
func (p *Params) ScanStrs(key string, in *[]string) *Params {
	v, ok := p.values[key]
	if !ok || v == nil || len(v) == 0 {
		*in = make([]string, 0)
		return p
	}
	*in = v
	return p
}
func (p *Params) ScanIntReq(key string, in *int) *Params {
	s := ""
	p.ScanStrReq(key, &s)
	if s == "" {
		return p
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		p.sate.err = errs.Merge(p.sate.err, errors.Wrapf(err, "param '%s' should be an integer", key))
		return p
	}
	*in = int(i)
	return p
}
func (p *Params) ScanInt(key string, in *int) *Params {
	s := ""
	p.ScanStr(key, &s)
	if s == "" {
		*in = 0
		return p
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		*in = 0
		return p
	}
	*in = int(i)
	return p
}
func (p *Params) ScanUint(key string, in *uint) *Params {
	s := ""
	p.ScanStr(key, &s)
	if s == "" {
		*in = 0
		return p
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		*in = 0
		return p
	}
	*in = uint(i)
	return p
}
func (p *Params) FinalErrState() error {
	err := p.sate.err
	p.ResetState()
	return err
}
func (p *Params) ResetState() *Params {
	p.sate = struct{ err error }{err: nil}
	return p
}

func httpParamsExec(r *http.Request) *Params {
	p := make(map[string][]string)
	for _, par := range []map[string][]string{r.URL.Query(), r.Form, r.PostForm} {
		for key, val := range par {
			p[key] = val
		}
	}
	op := make(map[string]interface{})
	if b, e := ioutil.ReadAll(r.Body); e == nil {
		_ = json.Unmarshal(b, &op)
	}
	for key, val := range op {
		p[key] = []string{fmt.Sprint(val)}
	}
	for key, val := range gmux.Vars(r) {
		p[key] = []string{val}
	}
	return &Params{
		values: p,
		file:   r.FormFile,
	}
}
