package rivet

import (
	"io"
	"net/http"
)

// HandleError 是 Rivet 缺省的错误处理方法
func HandleError(err error, rw http.ResponseWriter, req *http.Request) {
	if err == nil || err == io.EOF {
		return
	}
	code, ok := echo(err).(int)

	if !ok {
		code = http.StatusBadRequest
	}

	msg := err.Error()

	if msg == "" {
		msg = http.StatusText(code)
	}

	rw.WriteHeader(code)
	rw.Write([]byte(msg))
}
