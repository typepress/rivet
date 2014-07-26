package rivet

import (
	"net/http"
	"strings"
)

type router struct {
	rivet     Riveter
	notFounds *baseRoute
	get,
	patch,
	post,
	put,
	delete,
	options,
	head,
	any *table
}

/**
NewRouter 创建符合 Router 接口的实例.
参数:
	rivet 用于生成 Context 实例, 如果为 nil 使用 New() 创建一个.
*/
func NewRouter(rivet Riveter) Router {
	if rivet == nil {
		rivet = New()
	}
	r := &router{
		rivet:   rivet,
		get:     newTable(),
		patch:   newTable(),
		post:    newTable(),
		put:     newTable(),
		delete:  newTable(),
		options: newTable(),
		head:    newTable(),
		any:     newTable(),
	}
	return r
}

func (r *router) Get(pattern string, h ...Handler) Route {
	return r.addRoute("GET", pattern, h)
}

func (r *router) Post(pattern string, h ...Handler) Route {
	return r.addRoute("POST", pattern, h)
}

func (r *router) Put(pattern string, h ...Handler) Route {
	return r.addRoute("PUT", pattern, h)
}

func (r *router) Patch(pattern string, h ...Handler) Route {
	return r.addRoute("PATCH", pattern, h)
}

func (r *router) Delete(pattern string, h ...Handler) Route {
	return r.addRoute("DELETE", pattern, h)
}

func (r *router) Options(pattern string, h ...Handler) Route {
	return r.addRoute("OPTIONS", pattern, h)
}

func (r *router) Head(pattern string, h ...Handler) Route {
	return r.addRoute("HEAD", pattern, h)
}

func (r *router) Any(pattern string, h ...Handler) Route {
	return r.addRoute("*", pattern, h)
}

func (r *router) Add(method string, pattern string, h ...Handler) Route {
	return r.addRoute(method, pattern, h)
}

func (r *router) NotFound(h ...Handler) Route {
	route := &baseRoute{handlers: h}
	if h == nil {
		r.notFounds = nil
	} else {
		r.notFounds = route
	}
	return route
}

func (r *router) Rivet(rivet Riveter) {
	r.rivet = rivet
}

func (r *router) Match(urls []string, context Context) bool {
	var tab *table
	_, req := context.Source()
	switch req.Method {
	case "GET":
		tab = r.get
	case "PATCH":
		tab = r.patch
	case "POST":
		tab = r.post
	case "PUT":
		tab = r.put
	case "DELETE":
		tab = r.delete
	case "OPTIONS":
		tab = r.options
	case "HEAD":
		tab = r.head
	}

	if tab != nil && tab.Match(urls, context) ||
		r.any.Match(urls, context) {
		return true
	}

	if r.notFounds != nil {
		return r.notFounds.Match(urls, context)
	}
	return false
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var tab *table

	switch req.Method {
	case "GET":
		tab = r.get
	case "PATCH":
		tab = r.patch
	case "POST":
		tab = r.post
	case "PUT":
		tab = r.put
	case "DELETE":
		tab = r.delete
	case "OPTIONS":
		tab = r.options
	case "HEAD":
		tab = r.head
	}

	context := r.rivet.Context(res, req)
	urls := strings.Split(req.URL.Path, "/")
	if tab != nil &&
		(tab.MatchUrl(req.URL.Path, context) ||
			tab.Match(urls, context) ||
			r.any.MatchUrl(req.URL.Path, context) ||
			r.any.Match(urls, context)) {
		return
	}

	if r.notFounds != nil && r.notFounds.Match(urls, context) {
		return
	}

	http.NotFound(res, req)
}

func (r *router) addRoute(method string, pattern string, handlers []Handler) Route {
	var tab *table

	if pattern == "" {
		return r.NotFound(handlers...)
	}

	route := newRoute(pattern, handlers)
	switch method {
	default:
		tab = r.any
	case "GET":
		tab = r.get
	case "PATCH":
		tab = r.patch
	case "POST":
		tab = r.post
	case "PUT":
		tab = r.put
	case "DELETE":
		tab = r.delete
	case "OPTIONS":
		tab = r.options
	case "HEAD":
		tab = r.head
	}

	if route != nil {
		tab.addRoute(route)
		return route
	}

	s := &baseRoute{
		prefix:   pattern,
		handlers: handlers,
	}
	tab.literal[pattern] = s
	return s
}

// baseRoute
type baseRoute struct {
	/**
	prefix 前缀字符串
	用于静态路由和 NotFounds 时
		空值表示 NotFounds
		其它值表示静态 URL
	用于模式匹配时参见 route 说明
	*/
	prefix   string
	rivet    Riveter
	handlers []Handler
}

/**
route
维持 index 下标不变, 减少冗余数据.
prefix 为 pattern 的原值, 防止出现重复定义路由.
*/
type route struct {
	baseRoute
	pattern []*pattern // 模式匹配规则
	index   []uint8    // value 元素对应分割数组中的下标索引
	value   []string   // 固定字面值拼接字符串
	num     int        // urls 段数
	begin   int        // 保存同类 route 在 routes 中的开始位置
}

/**
两个 route 大小比较, 返回  -1, 0, 1, 表示 r 相对 z 的位置.
算法只是比较字面值, 因此返回 0 需要特别处理.
比较的次序要和匹配次序一直.
*/
func (r *route) cmp(z *route) int {

	// 段数比较
	if r.num < z.num {
		return -1
	}
	if r.num > z.num {
		return 1
	}

	// pattern 比较, 防止重复定义
	if r.prefix == z.prefix {
		return 0
	}

	// index/value 比较
	rn, zn := len(r.index), len(z.index)
	n := rn
	if zn < n {
		n = zn
	}

	ri, zi := r.index, z.index
	for i := 0; i < n; i++ {
		if ri[i] < zi[i] {
			return -1
		}
		if ri[i] > zi[i] {
			return 1
		}
	}

	// 部分 index 相等, 比较值
	rv, zv := r.value, z.value
	for i := 0; i < n; i++ {
		if rv[i] < zv[i] {
			return -1
		}
		if rv[i] > zv[i] {
			return 1
		}
	}

	if rn < zn {
		return -1
	}
	if rn > zn {
		return 1
	}

	return 0
}

/**
字面值都相等, 比较 pattern 前后缀, 用于插入位置,
现在的算法只能匹配第一个 pattern 前缀
*/
func (r *route) cmpattern(zp string) int {
	rp := r.pattern[0].prefix

	if rp == zp {
		return 0
	}
	if rp < zp {
		return -1
	}

	return 1
}

// newRoute 如果返回为 nil 表示静态路由, 由调用方处理
func newRoute(pattern string, handlers []Handler) (r *route) {
	// 字面路由
	pos := strings.IndexAny(pattern, ":*")
	if pos == -1 && -1 == strings.Index(pattern, "<") {
		return
	}

	urls := strings.Split(pattern, "/")
	if len(urls) == 0 || urls[0] != "" || len(urls) > 256 {
		panic(`rivet: invalide pattern`)
	}

	r = &route{}
	r.prefix = pattern
	r.handlers = handlers
	r.num = len(urls)

	for i, s := range urls {
		if i == 0 { // 第一个总是为 ""
			continue
		}

		p := newPattern(s)
		if p != nil {
			p.idx = uint8(i)
			r.pattern = append(r.pattern, p)
			continue
		}
		r.index = append(r.index, uint8(i))
		r.value = append(r.value, s)
	}

	return
}

func (r *baseRoute) Rivet(rivet Riveter) {
	r.rivet = rivet
}

// 字面路由和 NotFounds 路由
func (r *baseRoute) Match(urls []string, context Context) bool {
	if r.prefix != "" && urls != nil && r.prefix != strings.Join(urls, "/") {
		return false
	}

	// test
	if context == nil {
		return true
	}

	if r.rivet == nil {
		context.Invoke(nil, r.handlers...)
	} else {
		r.rivet.Context(context.Source()).Invoke(nil, r.handlers...)
	}
	return true
}

// Match 是给外部调用的, 内部需要优化二分法匹配

func (r *route) Match(urls []string, context Context) bool {
	if r.match(urls) != 0 {
		return false
	}

	return r.apply(urls, context)
}

/**
match 是内部调用的方法. 返回 -1,0, 1.
-1, 1 可用于二分法查找, 先确定有效范围.
0 表示可以用于数据匹配测试.
*/
func (r *route) match(urls []string) int {
	// 段数比较
	if r.num < len(urls) {
		return -1
	}
	if r.num > len(urls) {
		return 1
	}

	// 字面比较
	idx := uint8(len(r.index))

	var i uint8
	for ; i < idx; i++ {
		if r.value[i] < urls[r.index[i]] {
			return -1
		}
		if r.value[i] > urls[r.index[i]] {
			return 1
		}
	}

	return 0
}

// 模式匹配并调用 context.Invoke
func (r *route) apply(urls []string, context Context) bool {
	params := Params{}
	var v interface{}
	var ok bool
	for _, p := range r.pattern {
		v, ok = p.Match(urls[p.idx])
		if !ok {
			return false
		}
		if p.name != "" {
			params[p.name] = v
		}
	}

	if r.rivet == nil {
		context.Invoke(params, r.handlers...)
	} else {
		r.rivet.Context(context.Source()).Invoke(params, r.handlers...)
	}
	return true
}
