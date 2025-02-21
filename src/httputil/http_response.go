package httputil

import (
	"log"
	"net/http"
)

type HttpError struct {
	Msg  string
	Code int
}

type AppendResponseWriter struct {
	headers   http.Header
	Body      []byte
	Status    int
	separator []byte
}

func NewAppendResponseWriter() *AppendResponseWriter {
	return &AppendResponseWriter{headers: make(http.Header), separator: []byte("\n")}
}

func (arw *AppendResponseWriter) Header() http.Header {
	return arw.headers
}

func (arw *AppendResponseWriter) Write(body []byte) (int, error) {
	if len(arw.Body) > 0 {
		arw.Body = append(arw.Body, arw.separator...)
	}
	arw.Body = append(arw.Body, body...)
	return len(body), nil
}

func (arw *AppendResponseWriter) WriteHeader(status int) {
	arw.Status = status
}

type ObserverResponseWriter struct {
	Body   []byte
	Status int
	rw     http.ResponseWriter
}

func NewObserverResponseWriter(rw http.ResponseWriter) *ObserverResponseWriter {
	return &ObserverResponseWriter{rw: rw}
}

func (orw *ObserverResponseWriter) Header() http.Header {
	return orw.rw.Header()
}

func (orw *ObserverResponseWriter) Write(body []byte) (int, error) {
	orw.Body = append(orw.Body, body...)
	return orw.rw.Write(body)
}

func (orw *ObserverResponseWriter) WriteHeader(status int) {
	orw.Status = status
	orw.rw.WriteHeader(status)
}

func New500Error(msg string) *HttpError {
	return &HttpError{Code: http.StatusInternalServerError, Msg: msg}
}

func New400Error(msg string) *HttpError {
	return &HttpError{Code: http.StatusBadRequest, Msg: msg}
}

func RespondWithError(w http.ResponseWriter, err *HttpError) {
	log.Printf("Could not handle request: %s\n", err.Msg)
	http.Error(w, err.Msg, err.Code)
}
