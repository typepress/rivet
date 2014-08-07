package rivet

import (
	"io"
	"net/http"
	"reflect"
)

/**
New 返回一个 *Rivet, 实现了 Riveter, Context, Injector 接口.
事实上 New 返回值为 nil. 只能做 Riveter 使用.
此值在 http 请求期生成符合 Context, Injector 的实例.
*/
func New() *Rivet {
	return nil
}

var (
	id_httpRequest        = TypeIdOf((*http.Request)(nil))
	id_HttpResponseWriter = TypeIdOf((*ResponseWriter)(nil))
	id_ResponseWriter     = TypeIdOf((*http.ResponseWriter)(nil))
	id_Context            = TypeIdOf((*Context)(nil))
	id_Injector           = TypeIdOf((*Injector)(nil))
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

  // 获取 reflect.Type 的类型签名
	var rt *reflect.Type
	_ = TypeIdOf(rt) // reflect.Type 也是接口类型


这样获取的是 AnyType 的类型签名, 而不是 reflect.Type 的.

非接口类型:

	var s string
	_ = TypeIdOf(s)
	v := AnyNotInterfaceType{}
	_ = TypeIdOf(v)
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
	val    map[uint]interface{}
	res    ResponseWriter
	req    *http.Request
	params Params
	mapv   bool // 是否已经 Map 相关参数
}

/**
Context 生成 Context 实例, 此实例为 *Rivet.
如果参数 res 不符合 rivet.ResponseWriter 接口,
用 ResponseWriterFakeFlusher 包装一个.
*/
func (*Rivet) Context(res http.ResponseWriter, req *http.Request) Context {
	c := new(Rivet)
	c.res = NewResponseWriterFakeFlusher(res)
	c.req = req
	return c
}

/**
Source 返回构建 Context 的参数.
其中 http.ResponseWriter 实际是 rivet.ResponseWriter 实例,
有可能是 NewResponseWriterFakeFlusher 包装的.
*/
func (c *Rivet) Source() (http.ResponseWriter, *http.Request) {
	return c.res, c.req
}

func (c *Rivet) Request() *http.Request {
	return c.req
}

func (c *Rivet) Response() http.ResponseWriter {
	return c.res
}

func (c *Rivet) WriteString(data string) (int, error) {
	return io.WriteString(c.res, data)
}

func (c *Rivet) PathParams() Params {
	if c.params == nil {
		c.params = Params{}
	}
	return c.params
}

/**
Get 根据参数 t 表示的类型标识值, 从 context 中查找关联变量值.
如果未找到, 返回 nil.
*/
func (r *Rivet) Get(t uint) interface{} {
	switch t {
	case id_Context:
		t = id_Injector
	case id_HttpResponseWriter:
		t = id_ResponseWriter
	}

	return r.val[t]
}

/**
Map 等同 MapTo(v, TypeIdOf(v)). 以 v 的类型标识为 key.
Rivet 自动 Map 的变量类型有:
	Injector
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
MapTo 以 t 为 key 把变量 v 关联到 context. 相同 t 值只保留一个.
调用者也许会自己选择一个值, 注意选择值不能有类型标识冲突.
因为 Invoke 会自动从 Handler 函数中提取参数的类型标识,
如果 t 值不能对应某个类型, Invoke 也无法正确获取到变量.
可能现实中某些 v 是由调用者通过 Get(t) 获取, 和 Invoke 无关.
*/
func (r *Rivet) MapTo(v interface{}, t uint) {
	switch t {
	case id_Context:
		t = id_Injector
	case id_HttpResponseWriter:
		t = id_ResponseWriter
	}
	if r.val == nil {
		r.val = map[uint]interface{}{}
	}
	r.val[t] = v
}

/**
Invoke 遍历所有的 handler 函数. handler 函数返回值被忽略.
如果 handler 不是函数, 则被 Map 到 context.
如果 ResponseWriter.Written() 为 true, 终止遍历.
下列定义的 handler 函数被快速匹配:

	func()
	func(Context)
	func(Injector)
	func(*http.Request)
	func(ResponseWriter)
	func(http.ResponseWriter)
	func(ResponseWriter, *http.Request)
	func(http.ResponseWriter, *http.Request)
	func(Params)
	func(Params, *http.Request)
	func(Params, ResponseWriter)
	func(Params, http.ResponseWriter)
	func(Params, ResponseWriter, *http.Request)
	func(Params, http.ResponseWriter, *http.Request)
	func(Injector, ...Handler)
	func(Context, ...Handler)

最后两种含 Handler 参数的函数会接管 Invoke 控制权, Invoke 直接返回.
其他 handler 函数通过 reflect.Vlaue.Call 被调用.
Invoke 最后会执行 ResponseWriter.Flush().

注意:
	Invoke 对于没有进行 Map 的类型, 用 nil 替代.
	reflect.Vlaue.Call 可能产生 panic, 需要使用者处理.
*/
func (c *Rivet) Invoke(params Params, handler ...Handler) {
	var v reflect.Value

	if c.params == nil {
		c.params = params
	}

	for i, h := range handler {

		if c.res.Written() {
			break
		}

		switch fn := h.(type) {
		default: // 反射调用或者 Map 对象

			if !c.mapv {
				c.mapv = true
				c.MapTo(params, id_Params)
				c.MapTo(c, id_Injector)
				c.MapTo(c.req, id_httpRequest)
				c.MapTo(c.res, id_ResponseWriter)
			}

			v = reflect.ValueOf(h)
			if v.Kind() != reflect.Func {
				c.Map(h)
				continue
			}
			c.call(v)
		case func(Context):
			fn(c)
		case func(Injector):
			fn(c)

		case func(Injector, ...Handler): // 交接控制权
			fn(c, handler[i+1:]...)
			return
		case func(Context, ...Handler): // 交接控制权
			fn(c, handler[i+1:]...)
			return

		case func():
			fn()
		case func(ResponseWriter):
			fn(c.res)
		case func(http.ResponseWriter):
			fn(c.res)
		case func(*http.Request):
			fn(c.req)

		case func(ResponseWriter, *http.Request):
			fn(c.res, c.req)
		case func(http.ResponseWriter, *http.Request):
			fn(c.res, c.req)

		case func(Params):
			fn(params)

		case func(Params, ResponseWriter, *http.Request):
			fn(params, c.res, c.req)
		case func(Params, http.ResponseWriter, *http.Request):
			fn(params, c.res, c.req)

		case func(Params, ResponseWriter):
			fn(params, c.res)

		case func(Params, http.ResponseWriter):
			fn(params, c.res)

		case func(Params, *http.Request):
			fn(params, c.req)
		}
	}

	c.res.Flush()
}

/**
// 未确定是否增加此方法
func (c *Rivet) Call(function interface{}, args ...uint) {
	fn := reflect.ValueOf(function)
	if fn.Kind() != reflect.Func {
		panic("rivet: function are only allowed")
	}
	t := fn.Type()

	in := make([]reflect.Value, t.NumIn())

	if len(args) < len(in) {
		panic("rivet: arguments was not enough")
	}

	for i := 0; i < len(in); i++ {
		val := c.Get(args[i])
		in[i] = reflect.ValueOf(val)
	}

	if t.IsVariadic() {
		fn.CallSlice(in)
	} else {
		fn.Call(in)
	}
}
*/

func (c *Rivet) call(fn reflect.Value) {
	t := fn.Type()

	in := make([]reflect.Value, t.NumIn())
	for i := 0; i < len(in); i++ {
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
