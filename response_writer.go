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
	// Written 返回 Status()!=0 && Size()!=0 的结果
	Written() bool
}

/**
NewResponseWriterFakeFlusher 返回伪 http.Flusher 的 ResponseWriter 实例.
虽然含有 Flush() 方法, 但未实现 http.Flusher, 以后也不会实现.
这样设计提供了无 Flusher 的 ResponseWriter 并满足内置的 Rivet 接口要求.
如果您需要真的 http.Flusher 请不要使用此函数.
*/
func NewResponseWriterFakeFlusher(rw http.ResponseWriter) ResponseWriter {
	return &ResponseWriteFakeFlusher{rw, 0, 0}
}

/**
ResponseWriteFakeFlusher 实现了 http.ResponseWriter 接口和伪 http.Flusher 接口.
注意 Flush() 方法是个伪接口.
*/
type ResponseWriteFakeFlusher struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *ResponseWriteFakeFlusher) Flush() {}

func (rw *ResponseWriteFakeFlusher) WriteHeader(s int) {
	rw.ResponseWriter.WriteHeader(s)
	rw.status = s
}

func (rw *ResponseWriteFakeFlusher) Write(b []byte) (int, error) {
	if !rw.Written() {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *ResponseWriteFakeFlusher) Status() int {
	return rw.status
}

func (rw *ResponseWriteFakeFlusher) Size() int {
	return rw.size
}

func (rw *ResponseWriteFakeFlusher) Written() bool {
	return rw.status != 0
}
