package rivet

import (
	"net/http"
	"reflect"
)

/**
New 返回一个 *Rivet, 事实上值为 nil.
*/
func New() *Rivet {
	return nil
}

var (
	id_httpRequest        = TypeIdOf((*http.Request)(nil))
	id_HttpResponseWriter = TypeIdOf((*ResponseWriter)(nil))
	id_ResponseWriter     = TypeIdOf((*http.ResponseWriter)(nil))
	id_Context            = TypeIdOf((*Context)(nil))
	id_Rivet              = TypeIdOf((*Rivet)(nil))
	id_Params             = TypeIdOf(Params{})
)

/**
TypeIdOf 返回 v 的类型签名地址, 转换为 uint 类型.
此方法使用 reflect 获取 rType 的类型地址.

示例:
获取接口对象 v 的接口类型签名:

	// 获取 fmt.Stringer 接口类型签名
	var v *fmt.Stringer
	_ = TypeIdOf(v)
	// 或者
	_ = TypeIdOf((*fmt.Stringer)(nil))

获取 reflect.Type 的类型签名:

	var v AnyType
	_ = TypeIdOf(reflect.TypeOf(v))
这样获取的是 reflect.Type 的类型签名, 而不是 AnyType 的.

非接口类型:

	var s string
	_ = TypeIdOf(s)

*/
func TypeIdOf(v interface{}) uint {
	t, ok := v.(reflect.Type)
	if !ok {
		t = reflect.TypeOf(v)
	}
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Interface {
		t = t.Elem()
	}

	return uint(reflect.ValueOf(t).Pointer())
}

/**
Rivet 符合 Rivet, Context 接口.
*/
type Rivet struct {
	arg  map[uint]interface{} //[]*tv
	res  ResponseWriter
	req  *http.Request
	mapv bool // 是否已经 Map 相关参数
}

/**
Context 是 Rivet 的自构造方法.
如果参数 res 不符合 rivet.ResponseWriter 接口,
调用 NewResponseWriterFakeFlusher(res) 生成一个.
*/
func (*Rivet) Context(res http.ResponseWriter, req *http.Request) Context {
	rw, ok := res.(ResponseWriter)
	if !ok {
		rw = NewResponseWriterFakeFlusher(res)
	}

	c := new(Rivet)
	c.res = rw
	c.req = req
	c.arg = map[uint]interface{}{} //make([]*tv, 0, 20) //
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
	switch t {
	case id_Context:
		t = id_Rivet
	case id_HttpResponseWriter:
		t = id_ResponseWriter
	}

	return r.arg[t]
}

/**
Map 把变量值 v 关联到 context. 内部调用了 r.MapTo(v, T(v)).
Rivet 自动 Map 的类型有:
	*Rivet
	Context
	Params
	ResponseWriter
	http.ResponseWriter
	*http.Request
*/
func (r *Rivet) Map(v interface{}) {
	r.MapTo(v, TypeIdOf(v))
}

/**
MapTo 以类型值 t 把变量值 v 关联到 context.
相同类型的值只会保留一份.
*/
func (r *Rivet) MapTo(v interface{}, t uint) {
	switch t {
	case id_Context:
		t = id_Rivet
	case id_HttpResponseWriter:
		t = id_ResponseWriter
	}

	r.arg[t] = v
}

/**
Invoke 遍历所有的 handlers.
如果 handlers 是函数将被调用, 否则被 Map 到 context.
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
	func(Context)
	func(Context, Params)
	func(*Rivet, Params, ...Handler)
	func(Context, Params, ...Handler)

最后两种类型的 handler 会使 Invoke 直接结束遍历, handler 接管控制权.
其他 handler 会通过 reflect.Vlaue.Call 进行调用, handler 返回值被忽略.
Invoke 最后会执行 ResponseWriter.Flush().

注意:
	Invoke 对于没有进行 Map 的类型, 用 nil 替代.
	reflect.Vlaue.Call 可能产生 panic, 需要使用者处理.

*/
func (c *Rivet) Invoke(params Params, handlers ...Handler) {
	var v reflect.Value

	for i, h := range handlers {

		if c.res.Written() {
			break
		}

		switch fn := h.(type) {
		default: // 反射调用或者 Map 对象
			if !c.mapv {
				c.mapv = true
				c.MapTo(params, id_Params)
				c.MapTo(c, id_Rivet)
				c.MapTo(c.req, id_httpRequest)
				c.MapTo(c.res, id_ResponseWriter)
			}
			v = reflect.ValueOf(h)
			if v.Kind() != reflect.Func {
				c.Map(h)
				continue
			}
			c.call(v)

		case func(*Rivet, Params, ...Handler): // 交接控制权
			fn(c, params, handlers[i+1:]...)
			return
		case func(Context, Params, ...Handler): // 交接控制权
			fn(c, params, handlers[i+1:]...)
			return

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

		case func(ResponseWriter, *http.Request):
			fn(c.res, c.req)
			continue
		case func(http.ResponseWriter, *http.Request):
			fn(c.res, c.req)
			continue

		case func(Params):
			fn(params)
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

		case func(Context):
			fn(c)
			continue
		case func(Context, Params):
			fn(c, params)
			continue
		}
	}

	c.res.Flush()
}

func (c *Rivet) call(fn reflect.Value) {
	t := fn.Type()

	in := make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		id := TypeIdOf(t.In(i))
		val := c.Get(id)
		in[i] = reflect.ValueOf(val)
	}

	if t.IsVariadic() {
		fn.CallSlice(in)
	} else {
		fn.Call(in)
	}

}
