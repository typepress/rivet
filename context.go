package rivet

import (
	"io"
	"net/http"
	"reflect"
	"unsafe"
)

var (
	id_httpRequest    = TypePointerOf([]*http.Request{})
	id_ResponseWriter = TypePointerOf([]http.ResponseWriter{})
	id_Context        = TypePointerOf([]Context{})
	id_Params         = TypePointerOf([]Params{})
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

// Context 主要起到变量容器作用
type Context struct {
	Params
	Res  http.ResponseWriter
	Req  *http.Request
	Vars map[unsafe.Pointer]interface{} // 保存应用变量
}

// WriteString 是个便捷方法
func (c Context) WriteString(s string) (int, error) {
	return io.WriteString(c.Res, s)
}

// Var 返回类型指针 t 为键值的变量.
func (c Context) Var(t unsafe.Pointer) (v interface{}, has bool) {
	switch t {
	case id_Context:
		return c, true
	case id_httpRequest:
		return c.Req, true
	case id_ResponseWriter:
		return c.Res, true
	case id_Params:
		return c.Params, true
	}
	v, has = c.Vars[t]
	return
}

// Map 等同 MapTo(v, v).
func (c Context) Map(v interface{}) {
	c.Vars[TypePointerOf(v)] = v
}

// MapTo 以 TypePointerOf(t) 为键值保存变量 v. 相同类型值只保留一个.
func (c Context) MapTo(v interface{}, t interface{}) {
	c.Vars[TypePointerOf(t)] = v
}
