package rivet

import "net/url"

var BuildParameter = Lite

type Argument struct {
	Key     string      // 参数名
	Literal string      // 路径中的原始字符串
	Value   interface{} // 参数值, 字符串值转换后的值
}

func (a Argument) Name() string {
	return a.Key
}
func (a Argument) String() string {
	return a.Literal
}
func (a Argument) Val() interface{} {
	return a.Value
}

type Literal struct {
	Key   string // 参数名
	Value string // 路径中的原始字符串
}

func (a Literal) Name() string {
	return a.Key
}
func (a Literal) String() string {
	return a.Value
}
func (a Literal) Val() interface{} {
	return a.Value
}

// 参数值
type Parameter interface {
	Name() string
	String() string
	Val() interface{}
}

func Arg(name, literal string, val interface{}) Parameter {
	return Argument{Key: name, Literal: literal, Value: val}
}

func Lite(name, literal string, _ interface{}) Parameter {
	return Literal{Key: name, Value: literal}
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
