package rivet

import (
	"io"
	"net/http"
	"reflect"
	"unsafe"
)

var (
	idString = TypePointerOf([]string{})
	idBytes  = TypePointerOf([][]byte{})
	idError  = TypePointerOf([]error{})
	idBool   = TypePointerOf([]bool{})
)

// Dispatcher 接口用于派发
type Dispatcher interface {
	Dispatch(c Context) bool
}

type dispatchs []dispatch

func (ds dispatchs) Dispatch(c Context) bool {
	for _, d := range ds {
		if !d.Dispatch(c) {
			return false
		}
	}
	return true
}

type dispatch struct {
	fn         reflect.Value
	in         []unsafe.Pointer
	out        []unsafe.Pointer
	i          interface{}
	isVariadic bool
}

func (d dispatch) Dispatch(c Context) bool {
	var (
		v   interface{}
		has bool
		out []reflect.Value
	)

	if d.i != nil {
		switch fn := d.i.(type) {
		case func(Context):
			fn(c)

		case func(http.ResponseWriter, *http.Request):
			fn(c.Res, c.Req)

		case http.Handler:
			fn.ServeHTTP(c.Res, c.Req)

		case func(Params, http.ResponseWriter, *http.Request):
			fn(c.Params, c.Res, c.Req)

		case func(*http.Request):
			fn(c.Req)

		case func(http.ResponseWriter):
			fn(c.Res)

		case Dispatcher:
			return fn.Dispatch(c)
		case func():
			fn()
		default:
			c.Map(d.i)
		}
		return true
	}

	in := make([]reflect.Value, len(d.in))
	for i := 0; i < len(d.in); i++ {
		v, has = c.Pick(d.in[i])
		if !has {
			return false
		}

		in[i] = reflect.ValueOf(v)
	}

	if d.isVariadic {
		out = d.fn.CallSlice(in)
	} else {
		out = d.fn.Call(in)
	}

	if d.out == nil {
		return true
	}

	switch d.out[0] {
	case idString:
		io.WriteString(c.Res, out[0].String())

	case idBytes:
		c.Res.Write(out[0].Bytes())

	case idError:
		if !out[0].IsNil() {
			err, ok := out[0].Interface().(error)

			if ok {
				HandleError(err, c.Res, c.Req)
				return false
			}
		}
	case idBool:
		if !out[0].Bool() {
			return false
		}
	}

	if len(d.out) == 1 || out[1].IsNil() {
		return true
	}

	err, ok := out[1].Interface().(error)

	if ok && err != nil {
		HandleError(err, c.Res, c.Req)
		return false
	}

	return true
}

// Dispatch 包装 handler 为 Dispatcher
func Dispatch(handler ...interface{}) Dispatcher {
	var fun reflect.Value

	ds := make(dispatchs, 0)
	for _, i := range handler {
		if i == nil {
			continue
		}

		switch i.(type) {
		case
			func(),
			func(Context),
			func(*http.Request),
			func(http.ResponseWriter),
			func(http.ResponseWriter, *http.Request),
			func(Params, http.ResponseWriter, *http.Request),
			http.Handler, Dispatcher:

			ds = append(ds, dispatch{i: i})
			continue
		default:
			fun = reflect.ValueOf(i)
			if fun.Kind() != reflect.Func {
				ds = append(ds, dispatch{i: i})
				continue
			}
		}

		t := fun.Type()
		d := dispatch{
			fn:         fun,
			in:         make([]unsafe.Pointer, t.NumIn()),
			isVariadic: t.IsVariadic(),
		}

		for i := 0; i < t.NumIn(); i++ {
			d.in[i] = TypePointerOf(t.In(i))
		}

		if t.NumOut() > 0 && t.NumOut() <= 2 {

			out := make([]unsafe.Pointer, t.NumOut())

			for i := 0; i < t.NumOut(); i++ {
				out[i] = TypePointerOf(t.Out(i))
			}
			if out[0] == idError || out[0] == idString || out[0] == idBytes || out[0] == idBool ||
				len(out) == 2 && out[1] == idError {
				d.out = out
			}
		}

		ds = append(ds, d)
	}

	if len(ds) == 0 {
		return nil
	}

	return ds
}
