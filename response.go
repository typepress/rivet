package rivet

import (
	"net/http"
)

// Response 包装一个 http.ResponseWriter 对象并扩展了方法.
type Response struct {
	w      http.ResponseWriter
	status int
	size   int
}

// NewResponse 返回 *Response.
func NewResponse(rw http.ResponseWriter) *Response {
	return &Response{rw, 0, 0}
}

// Flush 实现 http.Flusher 接口方法.
// 如果原 http.ResponseWriter 实现了 http.Flusher 接口, 那么原 Flush() 方法会被调用.
func (r *Response) Flush() {
	if flusher, ok := r.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// WriteHeader 向相应发送状态码 s.
func (r *Response) WriteHeader(s int) {
	r.status = s
	r.w.WriteHeader(s)
}

// Write 向相应写入 b, 返回本次写入的字节和发生的错误.
func (r *Response) Write(b []byte) (int, error) {
	if !r.Written() {
		r.WriteHeader(http.StatusOK)
	}
	size, err := r.w.Write(b)
	r.size += size
	return size, err
}

// Status 返回通过 WriteHeader 设置的值.
func (r *Response) Status() int {
	return r.status
}

// Size 返回通过 Write 的总字节数.
func (r *Response) Size() int {
	return r.size
}

// Written 返回 true 表示通过 Write 或 WriteHeader 写入过, 否则返回 false.
func (r *Response) Written() bool {
	return r.status != 0 || r.size != 0
}
