package rivet

import (
	"strconv"
	"strings"
)

/**
FilterClass 保存 Fliter 生成器, 使用者可注册新的生成器.

内建 class 列表:

	*       Unicode characters, 允许空值, 等同于: ": *"
	string  非空 Unicode characters, 缺省值, 如果没有参数可省略.
	alpha   [a-zA-Z]+
	alnum   [a-zA-Z]+[0-9]+
	hex     [a-z0-9]+
	uint    uint 可以接收 strconv.ParseUint 的 bitSize 参数

其中: string, alpha, alnum, hex 都可以加最大长度限制参数, 如:
	":name string 10" 限制参数字符串字节长度不能超过 10
*/
var FilterClass = map[string]FilterBuilder{
	"*":      builtinFilter,
	"string": builtinFilter,
	"alpha":  builtinFilter,
	"alnum":  builtinFilter,
	"hex":    builtinFilter,
	"uint":   builtinFilter,
}

/**
NewFilter 通过访问 FilterClass 生成一个 Filter
如果 class 不存在或者生成 nil 将抛出 panic.
*/
func NewFilter(class string, args ...string) Filter {
	fn := FilterClass[class]
	if fn == nil {
		panic("rivet: not exists Filter class " + class)
	}
	p := fn(class, args...)
	if p == nil {
		panic("rivet: want an Filter, but got nil " + class)
	}
	return p
}

func builtinFilter(class string, args ...string) Filter {
	n := 0
	if len(args) != 0 {
		n, _ = strconv.Atoi(args[0])
	}
	switch class {
	case "*":
		return filterPass(true)
	case "string":
		return filterString(n)
	case "alpha":
		return filterAlpha(n)
	case "alnum":
		return filterAlnum(n)
	case "hex":
		return filterHex(n)
	case "uint":
		return filterUint(n)
	}
	return nil
}

type filterPass bool
type filterString int
type filterAlpha int
type filterUint int
type filterAlnum int
type filterHex int

func (n filterPass) Filter(s string) (interface{}, bool) {
	return s, true
}

func (n filterString) Filter(s string) (interface{}, bool) {
	if n != 0 && int(n) < len(s) {
		return nil, false
	}
	return s, len(s) != 0
}

func (n filterAlpha) Filter(s string) (interface{}, bool) {

	if n != 0 && int(n) < len(s) {
		return nil, false
	}

	for _, b := range []byte(s) {
		if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return nil, false
		}
	}
	return s, true
}

func (n filterUint) Filter(s string) (interface{}, bool) {

	i, err := strconv.ParseUint(s, 10, int(n))
	if err != nil {
		return nil, false
	}
	switch n {
	case 8:
		return uint8(i), true
	case 16:
		return uint16(i), true
	case 32:
		return uint32(i), true
	case 64:
		return i, true
	}
	return uint(i), true
}

func (n filterAlnum) Filter(s string) (interface{}, bool) {
	if n != 0 && int(n) < len(s) {
		return nil, false
	}

	a := []byte(s[1:])
	b := s[0]
	if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
		return nil, false
	}
	for _, b := range a {
		if (b < '0' || b > '9') && b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return nil, false
		}
	}
	return s, true
}

func (n filterHex) Filter(s string) (interface{}, bool) {
	if n != 0 && int(n) < len(s) {
		return nil, false
	}

	for _, b := range []byte(s) {
		if (b < '0' || b > '9') && b < 'a' || b > 'f' {
			return nil, false
		}
	}
	return s, true
}

// Node 专用
type perk struct {
	filter  Filter
	name    string // 空值匹配不提取
	noStyle bool   // 简化匹配
}

func newPerk(text string) *perk {
	if text[0] != ':' && text[0] != '*' {
		panic("rivet: internal error form newFilter : " + text)
	}

	a := strings.Split(text[1:], " ")

	p := new(perk)
	p.name = a[0]
	switch len(a) {
	case 1:
		p.noStyle = true
		if p.name == "" {
			p.filter = NewFilter("*")
		} else if p.name == "*" || p.name == ":" { // "/path/to/:pattern/to/**"
			p.name = "*"
			p.filter = NewFilter("*")
		} else {
			p.filter = NewFilter("string")
		}
	case 2:
		p.filter = NewFilter(a[1])
	default:
		p.filter = NewFilter(a[1], a[2:]...)
	}

	return p
}

func (p *perk) Perk(text string, params Params) bool {
	var ok bool
	var v interface{}

	if p.noStyle {
		if params != nil && p.name != "" {
			params[p.name] = text
		}
		return true
	}

	v, ok = p.filter.Filter(text)
	if ok && params != nil && p.name != "" {
		params[p.name] = v
	}
	return ok
}
