package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
)

type API struct{ s *application.Service }

func New(s *application.Service) *API { return &API{s: s} }
func (a *API) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/healthz", a.HandleHealth)
	m.HandleFunc("/api/v1/dossiers", a.HandleDossiers)
	m.HandleFunc("/api/v1/dossiers/", a.HandleDossier)
	return m
}
func (a *API) HandleHealth(w http.ResponseWriter, r *http.Request) {
	write(w, 200, map[string]any{"status": "ok"})
}

// maxBodyBytes 限制单次请求体长度。上限为 1 MiB。
const maxBodyBytes int64 = 1 << 20

// errRequestBodyTooLarge 在请求体超过 maxBodyBytes 上限时返回。
var errRequestBodyTooLarge = &domain.Error{Code: "REQUEST_BODY_TOO_LARGE", Message: "请求体超过1MiB上限"}

// readSingleJSONDocument 读取至多 maxBodyBytes 字节并校验请求体恰好包含一个
// 合法 JSON 文档。它区分真实 EOF 与受限读取器的截断 EOF：只有真实 EOF 且
// 第一个 JSON 文档之后仅有空白字节时才允许继续处理。若存在第二个 JSON 文档、
// 非空白尾随字节或任何超过上限的截断，返回稳定的客户端错误。
func readSingleJSONDocument(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	defer r.Body.Close()
	lr := io.LimitReader(r.Body, maxBodyBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, &applicationError{err: err}
	}
	if int64(len(data)) > maxBodyBytes {
		return nil, errRequestBodyTooLarge
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var v json.RawMessage
	if err := d.Decode(&v); err != nil {
		return nil, &applicationError{err: err}
	}
	rest := bytes.TrimLeft(data[d.InputOffset():], " \t\n\r\v\f")
	if len(rest) != 0 {
		return nil, &applicationError{err: io.ErrUnexpectedEOF}
	}
	return v, nil
}

func decode(r *http.Request, v any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &domain.Error{Code: "INVALID_CONTENT_TYPE", Message: "Content-Type必须为application/json"}
	}
	data, err := readSingleJSONDocument(r)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return &applicationError{err: err}
	}
	return nil
}

type applicationError struct{ err error }

func (e *applicationError) Error() string { return "请求JSON无效: " + e.err.Error() }

func strictUnmarshal(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return &applicationError{err: err}
	}
	rest := bytes.TrimLeft(data[d.InputOffset():], " \t\n\r\v\f")
	if len(rest) != 0 {
		return &applicationError{err: io.ErrUnexpectedEOF}
	}
	return nil
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func expected(r *http.Request) uint64 {
	v, _ := strconv.ParseUint(r.Header.Get("X-Expected-Version"), 10, 64)
	if v == 0 {
		v, _ = strconv.ParseUint(r.URL.Query().Get("expectedVersion"), 10, 64)
	}
	return v
}
func actor(r *http.Request) string {
	if x := r.Header.Get("X-Actor"); x != "" {
		return x
	}
	return "anonymous"
}
func key(r *http.Request) string { return r.Header.Get("Idempotency-Key") }
func errWrite(w http.ResponseWriter, e error) {
	if errors.Is(e, io.EOF) {
		writeError(w, &domain.Error{Code: "INVALID_JSON", Message: "请求体不能为空"})
		return
	}
	var appErr *applicationError
	if errors.As(e, &appErr) {
		writeError(w, &domain.Error{Code: "INVALID_JSON", Message: appErr.Error()})
		return
	}
	writeError(w, application.StateError(e))
}
func (a *API) HandleDossiers(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var in struct{ BuildingCode, Title, SurveyBoundary string }
		if e := decode(r, &in); e != nil {
			errWrite(w, e)
			return
		}
		d, e := a.s.Create(application.CreateInput{BuildingCode: in.BuildingCode, Title: in.Title, SurveyBoundary: in.SurveyBoundary}, key(r), actor(r))
		if e != nil {
			errWrite(w, e)
			return
		}
		write(w, 201, d)
		return
	}
	method(w)
}
func (a *API) HandleDossier(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dossiers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		write(w, 404, nil)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == "GET" {
		v, e := a.s.View(id)
		if e != nil {
			errWrite(w, e)
			return
		}
		write(w, 200, v)
		return
	}
	if len(parts) < 2 {
		write(w, 404, nil)
		return
	}
	if len(parts) != 2 {
		write(w, 404, nil)
		return
	}
	switch parts[1] {
	case "components":
		a.components(w, r, id)
	case "observations":
		a.observations(w, r, id)
	case "assess":
		a.assess(w, r, id)
	case "plans":
		a.plans(w, r, id)
	case "reviews":
		a.reviews(w, r, id)
	case "freeze":
		a.freeze(w, r, id)
	case "release":
		a.release(w, r, id)
	case "timeline":
		a.timeline(w, r, id)
	case "risk":
		a.risk(w, r, id)
	case "certificate":
		a.certificate(w, r, id)
	default:
		write(w, 404, nil)
	}
}
