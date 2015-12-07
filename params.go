package rivet

import "net/url"

// Argument
type Argument struct {
	Name   string      // 参数名
	Source string      // 参数原始字符串
	Value  interface{} // 参数值, 字符串值转换后的值, 如果就是字符串原值, 该值为 nil
}

// Params 保存从 URL.Path 中提取的参数.
type Params []Argument

// Get 返回第一个与 name 对应的字符串.
func (p Params) Get(name string) string {
	for _, a := range p {
		if a.Name == name {
			return a.Source
		}
	}

	return ""
}

// Get 返回第一个与 name 对应的值.
func (p Params) Value(name string) interface{} {
	for _, a := range p {
		if a.Name == name {
			if a.Value == nil {
				return a.Source
			}
			return a.Value
		}
	}

	return nil
}

// Gets 返回所有的原始字符串
func (p Params) Gets() map[string]string {
	m := make(map[string]string, len(p))
	for _, a := range p {
		m[a.Name] = a.Source
	}
	return m
}

// Values 返回所有的转换值
func (p Params) Values() map[string]interface{} {
	m := make(map[string]interface{}, len(p))
	for _, a := range p {
		if a.Value == nil {
			m[a.Name] = a.Source
		} else {
			m[a.Name] = a.Value
		}
	}
	return m
}

// AddTo 将字面值添加到 v.
func (p Params) AddTo(v url.Values) {
	for _, a := range p {
		v.Add(a.Name, a.Source)
	}
}
