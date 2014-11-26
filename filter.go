package rivet

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

/**
FilterClass 保存 Fliter 生成器, 使用者可注册新的生成器.

内建 class 列表:

	        空, 占位 class.
	*       Unicode characters, 允许空值, 等同于: ": *", 占位 class.
	string  非空 Unicode characters, 缺省值, 如果没有参数可省略.
	alpha   [a-zA-Z]+
	alnum   [a-zA-Z]+[0-9]+
	hex     [a-z0-9]+
	uint    uint 可以使用 strconv.ParseUint 的 bitSize 参数
	|       正则, 样例: ":id | ^id([0-9]+)$". 用 FindStringSubmatch 提取最后一个 Submatch.

其中: string, alpha, alnum, hex 可以加一个长度限制参数, 如:
	":name string 10" 限制参数字符串字节长度不超过 10.
*/
var FilterClass = map[string]FilterBuilder{
	"":       builtinFilter,
	"*":      builtinFilter,
	"string": builtinFilter,
	"alpha":  builtinFilter,
	"alnum":  builtinFilter,
	"hex":    builtinFilter,
	"uint":   builtinFilter,
	"|":      builtinFilter,
}

var empty string

/**
NewFilter 是缺省的 FilterBuilder.
通过调用 FilterClass 中与 class 对应的 FilterBuilder 生成一个 Filter.
如果相应的 FilterBuilder 或生成的 Filter 为 nil, 发生 panic.
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
	case "", "*":
		return filterTrue
	case "string":
		if n == 0 {
			return filterTrue
		}
		return filterString(n)
	case "alpha":
		return filterAlpha(n)
	case "alnum":
		return filterAlnum(n)
	case "hex":
		return filterHex(n)
	case "uint":
		return filterUint(n)
	case "|":
		return (*filterRegexp)(regexp.MustCompile(strings.Join(args, " ")))
	}
	return nil
}

// filterTrue 总是返回 true
var filterTrue = FilterFunc(
	func(s string) (string, interface{}, bool) {
		return s, s, true
	})

type filterString int
type filterAlpha int
type filterUint int
type filterAlnum int
type filterHex int
type filterRegexp regexp.Regexp

func (n filterString) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {

	if n != 0 && int(n) < len(s) {
		return empty, nil, false
	}
	return s, s, len(s) != 0
}

func (n filterAlpha) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {

	if n != 0 && int(n) < len(s) {
		return empty, nil, false
	}

	for _, b := range []byte(s) {
		if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return empty, nil, false
		}
	}
	return s, s, true
}

func (n filterUint) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {

	i, err := strconv.ParseUint(s, 10, int(n))
	if err != nil {
		return empty, nil, false
	}
	switch n {
	case 8:
		return s, uint8(i), true
	case 16:
		return s, uint16(i), true
	case 32:
		return s, uint32(i), true
	case 64:
		return s, i, true
	}
	return s, uint(i), true
}

func (n filterAlnum) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {

	if n != 0 && int(n) < len(s) {
		return empty, nil, false
	}

	a := []byte(s[1:])
	b := s[0]
	if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
		return empty, nil, false
	}
	for _, b := range a {
		if (b < '0' || b > '9') && b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return empty, nil, false
		}
	}
	return s, s, true
}

func (n filterHex) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {
	if n != 0 && int(n) < len(s) {
		return empty, nil, false
	}

	for _, b := range []byte(s) {
		if (b < '0' || b > '9') && b < 'a' || b > 'f' {
			return empty, nil, false
		}
	}
	return s, s, true
}

func (f *filterRegexp) Filter(s string,
	_ http.ResponseWriter, _ *http.Request) (string, interface{}, bool) {
	a := (*regexp.Regexp)(f).FindStringSubmatch(s)
	size := len(a)
	if size == 0 {
		return empty, nil, false
	}
	return a[size-1], a[size-1], true
}
