package rivet

import (
	"net/http"
)

/**
ResponseWriter 扩展 http.ResponseWriter.
*/
type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	// Status 返回调用 WriteHeader 设定的值, 初始值为 0
	Status() int
	// Size 返回调用 Write 写入的字节数, 初始值为 0
	Size() int
	// Written 返回 Status()!=0 && Size()!=0
	Written() bool
}

/**
NewResponseWriter 返回不支持 http.Flusher 的 ResponseWriter 实例.
虽然含有 Flush() 方法, 但未实现 http.Flusher, 以后也不会实现.
如果您需要 http.Flusher 请不要使用此函数.
*/
func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	return &responseWriter{rw, 0, 0}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) Flush() {}

func (rw *responseWriter) WriteHeader(s int) {
	rw.ResponseWriter.WriteHeader(s)
	rw.status = s
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.Written() {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) Size() int {
	return rw.size
}

func (rw *responseWriter) Written() bool {
	return rw.status != 0
}
