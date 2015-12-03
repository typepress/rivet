package rivet

import "net/url"

var BuildParameter = Source // 设置构建 Parameter 的方法

type argument struct {
	key    string      // 参数名
	source string      // 参数原始字符串
	value  interface{} // 参数值, 字符串值转换后的值
}

func (a argument) Name() string {
	return a.key
}
func (a argument) String() string {
	return a.source
}
func (a argument) Val() interface{} {
	return a.value
}

type sourceValue struct {
	key   string // 参数名
	value string // 路径中的原始字符串
}

func (s sourceValue) Name() string {
	return s.key
}
func (s sourceValue) String() string {
	return s.value
}
func (s sourceValue) Val() interface{} {
	return s.value
}

// 参数值
type Parameter interface {
	// Name 返回参数名.
	Name() string

	// String 返回参数的字符串形式值.
	String() string

	// Val 返回字符串值解析转换后的参数值.
	Val() interface{}
}

// Argument 返回以参数名, 字符串值和转换后的变量构建的 Parameter.
func Argument(name, source string, val interface{}) Parameter {
	return argument{name, source, val}
}

// Source 返回以参数名和字符串值构建的 Parameter.
func Source(name, source string, _ interface{}) Parameter {
	return sourceValue{name, source}
}

// Params 保存从 URL.Path 中提取的参数.
type Params []Parameter

// Get 返回第一个与 key 对应的字面值.
func (p Params) Get(key string) string {
	for _, a := range p {
		if a.Name() == key {
			return a.String()
		}
	}

	return ""
}

// Get 返回第一个与 key 对应的值.
func (p Params) Value(key string) interface{} {
	for _, a := range p {
		if a.Name() == key {
			return a.Val()
		}
	}

	return nil
}

// Gets 返回所有的原始字面值
func (p Params) Gets() map[string]string {
	m := make(map[string]string, len(p))
	for _, a := range p {
		m[a.Name()] = a.String()
	}
	return m
}

// Values 返回所有的转换值
func (p Params) Values() map[string]interface{} {
	m := make(map[string]interface{}, len(p))
	for _, a := range p {
		m[a.Name()] = a.Val()
	}
	return m
}

// AddTo 将字面值添加到 v.
func (p Params) AddTo(v url.Values) {
	for _, a := range p {
		v.Add(a.Name(), a.String())
	}
}
