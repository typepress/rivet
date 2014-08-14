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
	handler []interface{}
	mapv    bool // 是否已经 Map 相关参数
}

// NewContext 返回 *Rivet 实现的 Context
func NewContext(res http.ResponseWriter, req *http.Request) Context {

	c := new(Rivet)
	c.res = NewResponseWriterFakeFlusher(res)
	c.req = req
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

func (c *Rivet) ParamsReceiver(key, text string, val interface{}) {

	if c.params == nil {
		c.params = make(Params)
	}
	c.params[key] = val
}

// Handlers 设置 handler, 第一次使用有效.
func (c *Rivet) Handlers(handler ...interface{}) {
	if c.handler == nil {
		c.handler = handler
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

func (r *Rivet) get(t uint) (interface{}, bool) {

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
		return i, true
	}
	if t == id_MapStringInterface {
		return r.params, true
	}
	return nil, false
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
		r.val = make(map[uint]interface{})
	}
	r.val[t] = v
}

/**
Next 遍历 Handlers 保存的 handler, 通过 Invoke 调用.
如果 ResponseWriter.Written() 为 true, 终止遍历.
Next 最后会调用 ResponseWriter.Flush().
*/
func (c *Rivet) Next() {

	var h interface{}
	for {
		if len(c.handler) == 0 || c.res.Written() {
			break
		}
		h = c.handler[0]
		c.handler = c.handler[1:]
		c.Invoke(h)
	}

	c.res.Flush()
	c.handler = nil
}

/**
Invoke 处理 handler.

参数:
	handler 可以是任意值
		如果 handler 可被调用, 准备相应参数, 并调用 handler.
		否则 使用 Map 关联到 context.
返回:
	如果 handler 可被调用, 但是无法获取其参数, 返回 false.
	否则返回 true.

算法:
	如果 handler 是函数或者是有 ServeHTTP 方法的对象, 准备参数并调用.
	否则使用 Map 关联到 context.
	ServeHTTP 支持泛类型, 当然包括 http.Handler 实例.
	下列 handler 类型使用 switch 匹配, 参数直接传递, 未用 Get 从 context 获取:

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

注意:
	Invoke 未捕获可能产生的 panic, 需要使用者处理.
*/
func (c *Rivet) Invoke(handler interface{}) bool {

	switch fn := handler.(type) {
	default:

		// 反射调用或者 Map 对象
		fun := reflect.ValueOf(handler)
		if fun.Kind() != reflect.Func {
			fun = fun.MethodByName("ServeHTTP")
		}

		if fun.Kind() != reflect.Func {
			c.Map(handler)
			return true
		}

		t := fun.Type()

		in := make([]reflect.Value, t.NumIn())

		for i := 0; i < len(in); i++ {

			val, ok := c.get(TypeIdOf(t.In(i)))
			if !ok {
				return false
			}
			in[i] = reflect.ValueOf(val)
		}

		if t.IsVariadic() {
			fun.CallSlice(in)
		} else {
			fun.Call(in)
		}

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
	return true
}
