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
	// IsInjector 返回 true, 表示使用 Dispatch 方法派发, 否则使用 Hand 派发
	IsInjector() bool
	Dispatch(c *Context) bool
	Hand(Params, http.ResponseWriter, *http.Request) bool
}

type dispatchs struct {
	queue      []Dispatcher
	isInjector bool
}

func (ds dispatchs) IsInjector() bool { return ds.isInjector }
func (ds dispatchs) Dispatch(c *Context) bool {
	for _, d := range ds.queue {
		if d.IsInjector() {
			if !d.Dispatch(c) {
				return false
			}
		} else if !d.Hand(c.Params, c.Res, c.Req) {
			return false
		}
	}
	return true
}

func (ds dispatchs) Hand(p Params, rw http.ResponseWriter, req *http.Request) bool {
	for _, d := range ds.queue {
		if !d.IsInjector() && !d.Hand(p, rw, req) {
			return false
		}
	}
	return true
}

// 注入反射调用
type dispatcher struct {
	fn         reflect.Value
	in         []unsafe.Pointer
	out        []unsafe.Pointer
	isVariadic bool
}

func (d dispatcher) IsInjector() bool                                     { return true }
func (d dispatcher) Hand(Params, http.ResponseWriter, *http.Request) bool { return true }
func (d dispatcher) Dispatch(c *Context) bool {
	var (
		v   interface{}
		has bool
		out []reflect.Value
	)

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

type dispatchContext struct {
	c func(*Context)
	r func(*Context) bool
}

func (d dispatchContext) IsInjector() bool                                     { return true }
func (d dispatchContext) Hand(Params, http.ResponseWriter, *http.Request) bool { return true }
func (d dispatchContext) Dispatch(c *Context) bool {
	if d.c == nil {
		return d.r(c)
	}
	d.c(c)
	return true
}

type dispatchHandle struct {
	c func(http.ResponseWriter, *http.Request)
	r func(http.ResponseWriter, *http.Request) bool
}

func (d dispatchHandle) IsInjector() bool         { return false }
func (d dispatchHandle) Dispatch(_ *Context) bool { return true }
func (d dispatchHandle) Hand(_ Params, rw http.ResponseWriter, req *http.Request) bool {
	if d.c == nil {
		return d.r(rw, req)
	}
	d.c(rw, req)
	return true
}

type dispatchParams struct {
	c func(Params, http.ResponseWriter, *http.Request)
	r func(Params, http.ResponseWriter, *http.Request) bool
}

func (d dispatchParams) IsInjector() bool         { return false }
func (d dispatchParams) Dispatch(_ *Context) bool { return true }
func (d dispatchParams) Hand(p Params, rw http.ResponseWriter, req *http.Request) bool {
	if d.c == nil {
		return d.r(p, rw, req)
	}
	d.c(p, rw, req)
	return true
}

type dispatchHandler struct {
	http.Handler
}

func (d dispatchHandler) IsInjector() bool         { return false }
func (d dispatchHandler) Dispatch(_ *Context) bool { return true }
func (d dispatchHandler) Hand(_ Params, rw http.ResponseWriter, req *http.Request) bool {
	d.ServeHTTP(rw, req)
	return true
}

type dispatchEmpty func()

func (d dispatchEmpty) IsInjector() bool         { return false }
func (d dispatchEmpty) Dispatch(_ *Context) bool { return true }
func (d dispatchEmpty) Hand(_ Params, _ http.ResponseWriter, _ *http.Request) bool {
	d()
	return true
}

type dispatch struct {
	i interface{}
}

func (d dispatch) IsInjector() bool                                     { return true }
func (d dispatch) Hand(Params, http.ResponseWriter, *http.Request) bool { return true }
func (d dispatch) Dispatch(c *Context) bool {
	c.Map(d.i)
	return true
}

// ToDispatcher 包装 handler 为 Dispatcher.
// 特别的, 如果 handler 函数中包含 Store 类型参数
func ToDispatcher(handler ...interface{}) Dispatcher {
	var fun reflect.Value
	var withContext bool

	ds := make([]Dispatcher, 0)
	for _, i := range handler {
		if i == nil {
			continue
		}

		switch d := i.(type) {
		case func(*Context):
			withContext = true
			ds = append(ds, dispatchContext{c: d})
			continue
		case func():
			ds = append(ds, dispatchEmpty(d))
			continue
		case func(http.ResponseWriter, *http.Request):
			ds = append(ds, dispatchHandle{c: d})
			continue
		case func(Params, http.ResponseWriter, *http.Request):
			ds = append(ds, dispatchParams{c: d})
			continue

		case func(*Context) bool:
			withContext = true
			ds = append(ds, dispatchContext{r: d})
			continue
		case func(http.ResponseWriter, *http.Request) bool:
			ds = append(ds, dispatchHandle{r: d})
			continue
		case func(Params, http.ResponseWriter, *http.Request) bool:
			ds = append(ds, dispatchParams{r: d})
			continue

		// Dispatcher 接口优先于其它接口.
		case Dispatcher:
			if d.IsInjector() {
				withContext = true
			}
			ds = append(ds, d)
			continue
		case http.Handler:
			ds = append(ds, dispatchHandler{d})
			continue

		default:
			fun = reflect.ValueOf(i)
			if fun.Kind() != reflect.Func {
				ds = append(ds, dispatch{i})
				continue
			}
		}

		t := fun.Type()
		d := dispatcher{
			fn:         fun,
			in:         make([]unsafe.Pointer, t.NumIn()),
			isVariadic: t.IsVariadic(),
		}

		for i := 0; i < t.NumIn(); i++ {
			d.in[i] = TypePointerOf(t.In(i))
			if d.in[i] == idContext {
				withContext = true
			}
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
	if len(ds) == 1 {
		return ds[0]
	}
	return dispatchs{queue: ds, isInjector: withContext}
}
