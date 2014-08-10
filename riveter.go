package rivet

import (
	"io"
	"net/http"
	"reflect"
)

var (
	id_httpRequest        = TypeIdOf((*http.Request)(nil))
	id_HttpResponseWriter = TypeIdOf((*ResponseWriter)(nil))
	id_ResponseWriter     = TypeIdOf((*http.ResponseWriter)(nil))
	id_Context            = TypeIdOf((*Context)(nil))
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
Rivet 符合 Context 接口. 您应该用 NewContext 生成实例.
*/
type Rivet struct {
	val     map[uint]interface{}
	res     ResponseWriter
	req     *http.Request
	params  Params
	handler []Handler
	mapv    bool // 是否已经 Map 相关参数
}

// NewContext 返回 *Rivet 实现的 Context
func NewContext(res http.ResponseWriter, req *http.Request, params Params) Context {
	c := new(Rivet)
	c.res = NewResponseWriterFakeFlusher(res)
	c.req = req
	c.params = params
	return c
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
	return c.params
}

func (c *Rivet) Handlers(h ...Handler) {
	if c.handler == nil {
		c.handler = h
	}
}

/**
Get 根据参数 t 表示的类型标识值, 从 context 中查找关联变量值.
如果未找到, 返回 nil.
*/
func (r *Rivet) Get(t uint) interface{} {

	if !r.mapv {
		r.mapv = true
		r.MapTo(r.params, id_Params)
		r.MapTo(r, id_Context)
		r.MapTo(r.req, id_httpRequest)
		r.MapTo(r.res, id_ResponseWriter)
		r.MapTo(r.res, id_HttpResponseWriter)
	}

	return r.val[t]
}

/**
Map 等同 MapTo(v, TypeIdOf(v)). 以 v 的类型标识为 key.
Rivet 自动 Map 的变量类型有:
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
	if r.val == nil {
		r.val = map[uint]interface{}{}
	}
	r.val[t] = v
}

/**
Next 遍历所有的 handler 函数. handler 函数返回值被忽略.
如果 handler 不是函数, 则被 Map 到 context.
如果 ResponseWriter.Written() 为 true, 终止遍历.
下列定义的 handler 函数被快速匹配:

	func()
	func(Context)
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

Next 最后会执行 ResponseWriter.Flush().

注意:
	Next 对于没有进行 Map 的类型, 用 nil 替代.
	reflect.Vlaue.Call 可能产生 panic, 需要使用者处理.
*/
func (c *Rivet) Next() {
	var v reflect.Value
	var h interface{}

	for len(c.handler) > 0 {
		if c.res.Written() {
			break
		}
		h = c.handler[0]
		c.handler = c.handler[1:]

		switch fn := h.(type) {
		default: // 反射调用或者 Map 对象

			v = reflect.ValueOf(h)
			if v.Kind() != reflect.Func {
				c.Map(h)
				continue
			}
			c.call(v)
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
			fn(c.params)

		case func(Params, ResponseWriter, *http.Request):
			fn(c.params, c.res, c.req)
		case func(Params, http.ResponseWriter, *http.Request):
			fn(c.params, c.res, c.req)

		case func(Params, ResponseWriter):
			fn(c.params, c.res)

		case func(Params, http.ResponseWriter):
			fn(c.params, c.res)

		case func(Params, *http.Request):
			fn(c.params, c.req)
		}

	}

	c.res.Flush()
}

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
