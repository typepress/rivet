package rivet

import (
	"io"
	"net/http"
	"strconv"
)

const StatusNotFound = StatusError(http.StatusNotFound)
const StatusNotImplemented = StatusError(http.StatusNotImplemented)

type StatusError int

func (code StatusError) Error() string {
	return http.StatusText(int(code))
}

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

	if msg == "" {
		msg = "StatusError:" + strconv.Itoa(int(code))
	}

	rw.WriteHeader(code)
	rw.Write([]byte(msg))
}
