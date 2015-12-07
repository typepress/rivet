package rivet

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Matches 汇集以命名为 key 的 Matcher 生成器. 内建列表:
//
//  string  字符串, 缺省值.
//  alpha   [a-zA-Z]+
//  alnum   [a-zA-Z0-9]+
//  hex     [a-fA-F0-9]+
//  uint    可使用 strconv.ParseUint 进行转换, 支持 bitSize 参数
//  int     可使用 strconv.ParseInt 进行转换, 支持 bitSize 参数
//  reg     正则, 样例: ":id | ^id([0-9]+)$". 用 FindStringSubmatch 提取最后一个 Submatch.
//
// 其中: string, alpha, alnum, hex 可附加最小长度参数, 缺省值为 1.如:
//  ":name string 10" 限制参数字符串字节长度不超过 10.
var Matches = map[string]func(string) Matcher{
	"string": bString,
	"alpha":  bAlpha,
	"alnum":  bAlnum,
	"hex":    bHex,
	"uint":   bUint,
	"int":    bInt,
	"reg":    bRegexp,
}

var isOk = echo(echo)

// Ok 返回一个非 nil interface{}, 可被 Matcher 调用并返回该值, 表示匹配值就是原字符串,
// 那么 Trie 匹配到的 Argument.Vlaue 的值就是 nil. 目的是节省内存开销.
func Ok() interface{} {
	return isOk
}

// Matcher  用于匹配, 转换 URL.Path 参数.
type Matcher interface {

	// Matcher
	//
	// text 需要过滤的字符串, 例子:
	//   路由 "/blog/cat:id uint".
	//   路径 "/blog/cat3282".
	//   text 值是字符串 "3282".
	//
	// req 过滤器可能需要 Request 的信息.
	//
	// 返回值 val:
	//
	//   nil   匹配失败.
	//   Ok()  匹配成功且 text 作为返回值保存至 Argument.Source.
	//   其它  匹配成功, 保存至 Argument.Value.
	Match(text string, req *http.Request) (val interface{})
}

// builder 以 exp 首段字符串为名字, 从 Matches 创建一个 Matcher.
// 该名字用于排序, 大值优先执行. 该值不能为空.
func builder(exp string) Matcher {
	var m Matcher
	args := strings.SplitN(exp, " ", 2)

	if args[0] == "" {
		args = []string{"string", strings.TrimSpace(exp)}
	}

	build := Matches[args[0]]
	if build == nil {
		args = []string{"reg", exp}
		build = bRegexp
	}

	if build == nil {
		panic(fmt.Sprintf("rivet: not exists in Matches with %#v", exp))
	}

	if len(args) == 2 {
		m = build(args[1])
	} else {
		m = build(args[0])
	}

	if m == nil {
		panic(fmt.Sprintf("rivet: want an Matcher, but got nil with %#v", exp))
	}

	return m
}

// MatchFun 包装一个函数为 Matcher. 参数 fn, 可以是以下类型:
//
//   func(string) string
//   func(string) interface{}
//   func(string,*http.Request) string
//   func(string,*http.Request) interface{}
func MatchFun(fn interface{}) (m Matcher, ok bool) {
	ok = true
	switch fn := fn.(type) {
	case func(string) string:
		m = ssMatch(fn)
	case func(string) interface{}:
		m = siMatch(fn)
	case func(string, *http.Request) string:
		m = srsMatch(fn)
	case func(string, *http.Request) interface{}:
		m = sriMatch(fn)
	default:
		ok = false
	}
	return
}

type ssMatch func(string) string
type siMatch func(string) interface{}
type srsMatch func(string, *http.Request) string
type sriMatch func(string, *http.Request) interface{}

func (fn ssMatch) Match(text string, req *http.Request) interface{} {
	return fn(text)
}
func (fn siMatch) Match(text string, req *http.Request) interface{} {
	return fn(text)
}
func (fn srsMatch) Match(text string, req *http.Request) interface{} {
	return fn(text, req)
}
func (fn sriMatch) Match(text string, req *http.Request) interface{} {
	return fn(text, req)
}

type mString int
type mAlpha int
type mAlnum int
type mHex int
type mUint int
type mInt int
type mRegexp regexp.Regexp

func minOne(s string) int {
	if s == "" {
		return 1
	}
	n, _ := strconv.Atoi(s)
	return n
}

func bString(s string) Matcher {
	return mString(minOne(s))
}

func bAlpha(s string) Matcher {
	return mAlpha(minOne(s))
}

func bAlnum(s string) Matcher {
	return mAlnum(minOne(s))
}

func bHex(s string) Matcher {
	return mHex(minOne(s))
}

func bUint(s string) Matcher {
	return mUint(minOne(s))
}

func bInt(s string) Matcher {
	return mInt(minOne(s))
}

func bRegexp(s string) Matcher {
	return (*mRegexp)(regexp.MustCompile(s))
}

func (n mString) Match(s string, _ *http.Request) interface{} {
	if n != 0 && len(s) < int(n) {
		return nil
	}
	return isOk
}

func (n mAlpha) Match(s string, _ *http.Request) interface{} {

	if n != 0 && len(s) < int(n) {
		return nil
	}

	for _, b := range []byte(s) {
		if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return nil
		}
	}
	return isOk
}

func (n mAlnum) Match(s string, _ *http.Request) interface{} {

	if n != 0 && len(s) < int(n) {
		return nil
	}

	a := []byte(s[1:])
	b := s[0]
	if b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
		return nil
	}
	for _, b := range a {
		if (b < '0' || b > '9') && b < 'A' || b > 'z' || (b > 'Z' && b < 'a') {
			return nil
		}
	}
	return isOk
}

func (n mUint) Match(s string, _ *http.Request) interface{} {
	i, err := strconv.ParseUint(s, 10, int(n))
	if err != nil {
		return nil
	}
	switch n {
	case 8:
		return uint8(i)
	case 16:
		return uint16(i)
	case 32:
		return uint32(i)
	case 64:
		return uint64(i)
	}
	return i
}

func (n mInt) Match(s string, _ *http.Request) interface{} {

	i, err := strconv.ParseInt(s, 10, int(n))
	if err != nil {
		return nil
	}
	switch n {
	case 8:
		return int8(i)
	case 16:
		return int16(i)
	case 32:
		return int32(i)
	case 64:
		return int64(i)
	}
	return i
}

func (n mHex) Match(s string, _ *http.Request) interface{} {

	if n != 0 && len(s) < int(n) {
		return nil
	}

	for _, b := range []byte(s) {
		if (b < '0' || b > '9') && b < 'a' || b > 'f' {
			return nil
		}
	}
	return isOk
}

func (f *mRegexp) Match(s string, _ *http.Request) interface{} {
	a := (*regexp.Regexp)(f).FindStringSubmatch(s)
	size := len(a)
	if size == 0 {
		return nil
	}
	return a[size-1]
}
