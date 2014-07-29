package rivet

import (
	"net/http"
	"reflect"
	"sort"
	"unsafe"
)

/**
New 返回一个 *Rivet, 事实上值为 nil.
*/
func New() *Rivet {
	return (*Rivet)(nil)
}

/**
Rivet 符合 Rivet, Context 接口.
*/
type Rivet struct {
	arg []*tv
	res ResponseWriter
	req *http.Request
	mp  bool // 是否已经 Map(params)
}

/**
T 返回 i 所属类型的 uint 标识值.
此方法跟 Go 内部实现有关, 需要同步跟进.
*/
func T(i interface{}) uint {
	return *(*uint)(unsafe.Pointer(&i))
}

// 专门从 reflect.Type 取原类型标识值.
func rTypeT(t reflect.Type) uint {
	e := *(*reflectTypeInterface)(unsafe.Pointer(&t))
	return e.addrs
}

type reflectTypeInterface struct {
	rtype *uint
	addrs uint
}

type tv struct {
	t uint        // type
	v interface{} // val
}

/**
Context 是 Rivet 的自构造方法.
如果参数 res 不符合 rivet.ResponseWriter 接口, 调用 NewResponseWriter(res) 生成一个.
*/
func (*Rivet) Context(res http.ResponseWriter, req *http.Request) Context {

	rw, ok := res.(ResponseWriter)
	if !ok {
		rw = NewResponseWriter(res)
	}
	c := &Rivet{
		res: rw,
		req: req,
		arg: make([]*tv, 0, 20),
	}
	c.Map(res)
	c.Map(rw)
	c.Map(req)
	c.Map(c)
	return c
}

func (c *Rivet) Source() (http.ResponseWriter, *http.Request) {
	return c.res, c.req
}

/**
Get 根据参数 t 表示的类型标识值, 从 context 中查找关联变量值.
如果未找到, 返回 nil.
*/
func (r *Rivet) Get(t uint) interface{} {
	n := len(r.arg)
	if n == 0 {
		return nil
	}
	pos := sort.Search(n, func(i int) bool {
		return r.arg[i].t >= t
	})
	if pos == n || r.arg[pos].t != t {
		return nil
	}
	return r.arg[pos].v
}

/**
Map 把变量值 v 关联到 context. 内部调用了 r.MapTo(v, T(v)).
Rivet 自动 Map 的 context 相关变量值类型有:
	*Rivet
	Params
	ResponseWriter
	http.ResponseWriter
	*http.Request
*/
func (r *Rivet) Map(v interface{}) {
	r.MapTo(v, T(v))
}

/**
MapTo 以类型值 t 把变量值 v 关联到 context.
相同类型的值只会保留一份.
*/
func (r *Rivet) MapTo(v interface{}, t uint) {
	n := len(r.arg)
	a := &tv{t: t, v: v}

	if n == 0 {
		r.arg = append(r.arg, a)
		return
	}

	pos := sort.Search(n, func(i int) bool {
		return r.arg[i].t > t
	})

	if pos == n {
		r.arg = append(r.arg, a)
		return
	}

	if r.arg[pos].t == t {
		r.arg[pos] = a
		return
	}
	r.arg = append(r.arg, nil)
	for i := n; i > pos; i-- {
		r.arg[i] = r.arg[i-1]
	}
	r.arg[pos] = a
}

/**
Invoke 遍历所有的 handlers.
如果 handlers 是函数将被调用, 否则被 Map 关联到 context.
如果 ResponseWriter.Written() 为 true, 终止遍历.
下列定义的 handler 被快速匹配:

	func()
	func(ResponseWriter)
	func(http.ResponseWriter)
	func(*http.Request)
	func(Params)
	func(ResponseWriter, *http.Request)
	func(http.ResponseWriter, *http.Request)
	func(ResponseWriter, *http.Request, Params)
	func(http.ResponseWriter, *http.Request, Params)
	func(ResponseWriter, Params)
	func(http.ResponseWriter, Params)
	func(*http.Request, Params)
	func(*Rivet)
	func(*Rivet, Params)
	func(*Rivet, Params, ...Handler)

定义为 func(*Rivet, Params, ...Handler) 的 handler 会接手控制权.
其他 handler 会通过 reflect.Vlaue.Call 进行调用, handler 返回值被忽略.
Invoke 最后会执行 ResponseWriter.Flush().

注意:
	NewResponseWriter 产生的实例未实现 http.Flusher.
	Invoke 对于没有进行 Map 的类型, 用 nil 替代.
	reflect.Vlaue.Call 可能产生 panic, 需要使用者处理.
*/
func (c *Rivet) Invoke(params Params, handlers ...Handler) {
	if !c.mp {
		c.mp = true
		c.Map(params)
	}
	var t reflect.Type
	for i, h := range handlers {

		if c.res.Written() {
			break
		}

		switch fn := h.(type) {
		case func():
			fn()
			continue
		case func(ResponseWriter):
			fn(c.res)
			continue
		case func(http.ResponseWriter):
			fn(c.res)
			continue
		case func(*http.Request):
			fn(c.req)
			continue
		case func(Params):
			fn(params)
			continue

		case func(ResponseWriter, *http.Request):
			fn(c.res, c.req)
			continue
		case func(http.ResponseWriter, *http.Request):
			fn(c.res, c.req)
			continue

		case func(ResponseWriter, *http.Request, Params):
			fn(c.res, c.req, params)
			continue
		case func(http.ResponseWriter, *http.Request, Params):
			fn(c.res, c.req, params)
			continue

		case func(ResponseWriter, Params):
			fn(c.res, params)
			continue
		case func(http.ResponseWriter, Params):
			fn(c.res, params)
			continue

		case func(*http.Request, Params):
			fn(c.req, params)
			continue
		case func(*Rivet):
			fn(c)
			continue
		case func(*Rivet, Params):
			fn(c, params)
			continue
		case func(*Rivet, Params, ...Handler): // 交接控制权
			fn(c, params, handlers[i+1:]...)
			return
		default:
			t = reflect.TypeOf(h)
			if t.Kind() != reflect.Func {
				c.Map(h)
				continue
			}
		}
		c.call(t, h)
	}

	c.res.Flush()
}

func (c *Rivet) call(t reflect.Type, h interface{}) {
	in := make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		id := rTypeT(t.In(i))
		val := c.Get(id)
		in[i] = reflect.ValueOf(val)
	}

	reflect.ValueOf(h).Call(in)
}
