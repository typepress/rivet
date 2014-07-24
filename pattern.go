package rivet

import (
	"strconv"
	"strings"
)

/**
PatternFactory 用于注册 Pattern 实例工厂, 内建的 class 有
	*       Unicode characters, 允许空值, 等同于: < *>
	string  非空 Unicode characters, 缺省值, 如果没有参数可省略.
	alpha   [a-zA-Z]+
	alnum   [a-zA-Z]+[0-9]+
	hex     [a-z0-9]+
	uint    uint 可以接收 strconv.ParseUint 的 bitSize 参数
注意: <name string 0> 中的 0 无法产生作用, 应该用 <name *> 替代.
*/
var PatternFactory = map[string]func(class string, args ...string) Pattern{
	"*":      patternBuiltin,
	"string": patternBuiltin,
	"alpha":  patternBuiltin,
	"alnum":  patternBuiltin,
	"hex":    patternBuiltin,
	"uint":   patternBuiltin,
}

/**
NewPattern 通过访问 PatternFactory 生成一个 Pattern
如果 class 不存在或者生成 nil 将抛出 panic.
*/
func NewPattern(class string, args ...string) Pattern {
	fn := PatternFactory[class]
	if fn == nil {
		panic("rivet: not exists Pattern class " + class)
	}
	p := fn(class, args...)
	if p == nil {
		panic("rivet: want an Pattern, but got nil " + class)
	}
	return p
}

func patternBuiltin(class string, args ...string) Pattern {
	n := 0
	if len(args) != 0 {
		n, _ = strconv.Atoi(args[0])
	}
	switch class {
	case "*":
		return patternPass(true)
	case "string":
		return patternString(n)
	case "alpha":
		return patternAlpha(n)
	case "alnum":
		return patternAlnum(n)
	case "hex":
		return patternHex(n)
	case "uint":
		return patternUint(n)
	}
	return nil
}

type patternPass bool
type patternString int
type patternAlpha int
type patternUint int
type patternAlnum int
type patternHex int

func (n patternPass) Match(s string) (interface{}, bool) {
	return s, true
}

func (n patternString) Match(s string) (interface{}, bool) {
	if n != 0 && int(n) < len(s) {
		return nil, false
	}
	return s, len(s) != 0
}

func (n patternAlpha) Match(s string) (interface{}, bool) {
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

func (n patternUint) Match(s string) (interface{}, bool) {

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

func (n patternAlnum) Match(s string) (interface{}, bool) {
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

func (n patternHex) Match(s string) (interface{}, bool) {
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

// route 匹配使用的 pattern
type pattern struct {
	Pattern
	name           string // 全空表示 "*"
	prefix, suffix string // 前后缀
}

func newPattern(s string) *pattern {
	pos := strings.Index(s, ":")
	if pos != -1 {
		p := &pattern{}
		p.prefix = s[:pos]
		p.name = s[pos+1:]
		p.Pattern = NewPattern("string")
		return p
	}

	pos = strings.Index(s, "*")
	if pos != -1 {
		p := &pattern{}
		p.prefix = s[:pos]
		p.suffix = s[pos+1:]
		p.Pattern = NewPattern("*")
		return p
	}

	pos = strings.Index(s, "<")
	if pos != -1 {
		end := strings.Index(s, ">")
		a := strings.Split(s[pos+1:end], " ")
		if len(a) == 0 {
			panic(`rivet: invalide pattern ` + s)
		}
		p := &pattern{}
		p.prefix = s[:pos]
		p.suffix = s[end+1:]
		p.name = a[0]
		if len(a) == 1 {
			p.Pattern = NewPattern("string")
		} else {
			p.Pattern = NewPattern(a[1], a[1:]...)
		}
		return p
	}
	return nil
}

func (p *pattern) Match(s string) (interface{}, bool) {

	n := len(p.prefix)
	if n != 0 {
		if len(s) < n || p.prefix != s[0:n] {
			return nil, false
		}
		s = s[n:]
	}
	n = len(p.suffix)
	if n != 0 {
		n = len(s) - n
		if n < 0 || p.suffix != s[n:] {
			return nil, false
		}
		s = s[0:n]
	}
	return p.Pattern.Match(s)
}
