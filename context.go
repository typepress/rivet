package rivet

import (
	"io"
	"net/http"
	"reflect"
	"unsafe"
)

var (
	idRequest        = TypePointerOf([]*http.Request{})
	idResponseWriter = TypePointerOf([]http.ResponseWriter{})
	idContext        = TypePointerOf([]*Context{})
	idParams         = TypePointerOf([]Params{})
)

// emptyInterface 是 interface{} 的通用结构. 参见 reflect:emptyInterface.
type emptyInterface struct {
	Type unsafe.Pointer
	Word unsafe.Pointer
}

// TypePointerOf 返回 unsafe.Pointer 表示的类型指针. 该方法有固定的使用方法.
// 假设要返回变量 V 的类型 T 的指针,
//
//   TypePointerOf(V)                  // V 非 nil 且 T 的风格不是 Slice.
//   TypePointerOf(reflect.TypeOf(V))  // V 非 nil
//   TypePointerOf(reflect.ValueOf(V)) // V 非 nil
//   TypePointerOf([]T{})              // 通用形式
func TypePointerOf(i interface{}) unsafe.Pointer {
	if i == nil {
		return nil
	}

	switch v := i.(type) {
	case reflect.Type:
		return (*emptyInterface)(unsafe.Pointer(&i)).Word
	case reflect.Value:
		i = v.Type()
		return (*emptyInterface)(unsafe.Pointer(&i)).Type
	}

	t := reflect.TypeOf(i)

	if t.Kind() == reflect.Slice {
		i = t.Elem()
		return (*emptyInterface)(unsafe.Pointer(&i)).Word
	}

	return (*emptyInterface)(unsafe.Pointer(&i)).Type
}

// Context 主要是注入变量的容器.
type Context struct {
	Params
	Res http.ResponseWriter
	Req *http.Request

	// Store 是个简单的数据容器, 以字符串为 Key 存储变量.
	// 当不需要反射调用时, 使用 Store 更轻量.
	// 使用前您需要先 make 它.
	Store   map[string]interface{}
	partner map[unsafe.Pointer]interface{} // 保存响应期关联变量
}

// Pick 返回类型指针 t 为键值的关联变量.
// 如果 t 表示 Context, Params, http.ResponseWriter, *http.Request 类型,
// Pick 直接返回 c 或者相应成员, 否则返回 MapTo 关联的变量.
func (c *Context) Pick(t unsafe.Pointer) (v interface{}, ok bool) {
	switch t {
	case idContext:
		return c, true
	case idRequest:
		return c.Req, true
	case idResponseWriter:
		return c.Res, true
	case idParams:
		return c.Params, true
	}
	if c.partner != nil {
		v, ok = c.partner[t]
	}
	return
}

// Map 等同 MapTo(v, v).
func (c *Context) Map(v interface{}) {
	c.MapTo(v, v)
}

// MapTo 以 TypePointerOf(t) 为键值把变量 v 关联到 context. 相同 t 值只保留一个.
// 无需保存 Context, Params, http.ResponseWriter, *http.Request 类型变量, 参见 Pick.
func (c *Context) MapTo(v interface{}, t interface{}) {
	if c.partner == nil {
		c.partner = make(map[unsafe.Pointer]interface{}, 1)
	}
	c.partner[TypePointerOf(t)] = v
}

// WriteString 是个便捷方法
func (c *Context) WriteString(s string) (int, error) {
	return io.WriteString(c.Res, s)
}
