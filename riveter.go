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
	id_MapStringInterface = TypeIdOf(map[string]interface{}{})
)

/**
TypeIdOf 返回 v 的类型签名地址, 转换为 uint 类型.
此方法使用 reflect 获取类型地址.

示例:
获取 fmt.Stringer 接口类型签名:

	var v *fmt.Stringer
	_ = TypeIdOf(v)
	// 或者
	_ = TypeIdOf((*fmt.Stringer)(nil))

获取 reflect.Type 本身的类型签名:

	var rt *reflect.Type
	_ = TypeIdOf(rt) // reflect.Type 也是接口类型
	// 或者
	t := reflect.TypeOf(nil)
	_ = TypeIdOf(&t)

获取函数的参数类型签名:

	t := reflect.TypeOf(fmt.Println)
	_ = TypeIdOf(t.In(0))

非接口类型:

	var s string
	_ = TypeIdOf(s) // 等同 TypeIdOf("")
	var v AnyNotInterfaceType
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

// Request 返回生成 Context 的 *http.Request
func (c *Rivet) Request() *http.Request {
	return c.req
}

// Response 返回生成 Context 的 http.ResponseWriter
func (c *Rivet) Response() http.ResponseWriter {
	return c.res
}

// WriteString 方便向 http.ResponseWriter 写入 string.
func (c *Rivet) WriteString(data string) (int, error) {
	return io.WriteString(c.res, data)
}

//	Params 返回路由匹配时从 URL.Path 中提取的参数
func (c *Rivet) Params() Params {
	return c.params
}

// Handlers 设置 Handler, 第一次使用有效.
func (c *Rivet) Handlers(h ...Handler) {
	if c.handler == nil {
		c.handler = h
	}
}

/**
Get 以类型标识 t 为 key, 获取关联到 context 的变量.
如果未找到, 返回 nil.
特别的如果函数参数用了 map[string]interface{}, 且 Get 为 nil, 用 Params 代替.
这样做, 如果不用 Map 功能, 所写的 Handler 就不需要 import rivet.
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
	i, ok := r.val[t]
	if ok {
		return i
	}
	if t == id_MapStringInterface {
		return r.params
	}
	return nil
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
调用者也许会自己定义一个值, 注意选择值不能和真实类型标识冲突.
否则可能会传递给 Handler 错误的参数.
*/
func (r *Rivet) MapTo(v interface{}, t uint) {
	if r.val == nil {
		r.val = map[uint]interface{}{}
	}
	r.val[t] = v
}

/**
Next 遍历调用 handler, handler 返回值被忽略.
如果 handler 不是函数也不含 ServeHTTP 方法, 使用 Map 关联到 context.
ServeHTTP 只是方法的名字, 支持泛类型, 当然包括 http.Handler.
如果 ResponseWriter.Written() 为 true, 终止遍历.
下列 handler 被直接匹配, 参数直接传递, 未用 Get 从 context 获取:

	func()
	func(Context)
	func(*http.Request)
	func(ResponseWriter)
	func(ResponseWriter, *http.Request)
	func(http.ResponseWriter)
	func(http.ResponseWriter, *http.Request)
	func(Params)
	func(Params, *http.Request)
	func(Params, ResponseWriter)
	func(Params, http.ResponseWriter)
	func(Params, ResponseWriter, *http.Request)
	func(Params, http.ResponseWriter, *http.Request)
	http.Handler

Next 最后会执行 ResponseWriter.Flush().

注意:
	Next 对于没有进行 Map 的类型, 用 nil 替代.
	Next 未捕获调用 handler 可能产生的 panic, 需要使用者处理.
*/
func (c *Rivet) Next() {
	var h interface{}

	for len(c.handler) > 0 {
		if c.res.Written() {
			break
		}
		h = c.handler[0]
		c.handler = c.handler[1:]

		switch fn := h.(type) {
		default:
			// 反射调用或者 Map 对象
			c.call(h)
		case http.Handler:
			fn.ServeHTTP(c.res, c.req)
		case func():
			fn()
		case func(Context):
			fn(c)

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

func (c *Rivet) call(h interface{}) {

	fn := reflect.ValueOf(h)
	if fn.Kind() != reflect.Func {
		fn = fn.MethodByName("ServeHTTP")
	}

	if fn.Kind() != reflect.Func {
		c.Map(h)
		return
	}

	t := fn.Type()

	in := make([]reflect.Value, t.NumIn())

	for i := 0; i < len(in); i++ {
		id := TypeIdOf(t.In(i))
		val := c.Get(id)

		/** panic ???
		if val == nil {
			panic("rivet: value not found of " + t.In(i).String())
		}
		//*/

		in[i] = reflect.ValueOf(val)
	}

	if t.IsVariadic() {
		fn.CallSlice(in)
	} else {
		fn.Call(in)
	}

}
