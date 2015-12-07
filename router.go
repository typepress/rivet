package rivet

import (
	"net/http"
	"strings"
)

// Router 管理路由.
type Router map[string]*Trie

func echo(i interface{}) interface{} {
	return i
}

// Match 匹配路由节点. 返回值参见 Trie.Match.
// 行为:
//
//   在 "HEAD" 方法中匹配不到时, 尝试在 "GET" 方法中匹配.
//   最后尝试在 "any" 方法中匹配.
//
// 参数:
//
// 	method   Request.Method, "*" 等同 "any"
// 	urlPath  Request.URL.Path, 缺省为 "/".
// 	rw       http 响应, 传递给 Trie.
// 	req      http 请求, 传递给 Trie.
func (r Router) Match(method, urlPath string, req *http.Request) (t *Trie, params Params, err error) {
	t = r[method]

	if method == "*" {
		method = "any"
	}

	if urlPath == "" {
		urlPath = "/"
	}

	if t != nil {
		t, params, err = t.Match(urlPath, req)
	}

	if err == nil && t == nil && method == "HEAD" {
		if t = r["GET"]; t != nil {
			t, params, err = t.Match(urlPath, req)
		}
	}

	if err == nil && t == nil && method != "any" {
		if t = r["any"]; t != nil {
			t, params, err = t.Match(urlPath, req)
		}
	}
	return
}

// Get 为 HTTP GET request 设置路由
func (r Router) Get(pattern string, handler ...interface{}) *Trie {
	return r.Handle("GET", pattern, handler...)
}

// Post 为 HTTP POST request 设置路由
func (r Router) Post(pattern string, handler ...interface{}) *Trie {
	return r.Handle("POST", pattern, handler...)
}

// Put 为 HTTP PUT request 设置路由
func (r Router) Put(pattern string, handler ...interface{}) *Trie {
	return r.Handle("PUT", pattern, handler...)
}

// Patch 为 HTTP PATCH request 设置路由
func (r Router) Patch(pattern string, handler ...interface{}) *Trie {
	return r.Handle("PATCH", pattern, handler...)
}

// Delete 为 HTTP DELETE request 设置路由
func (r Router) Delete(pattern string, handler ...interface{}) *Trie {
	return r.Handle("DELETE", pattern, handler...)
}

// Options 为 HTTP OPTIONS request 设置路由
func (r Router) Options(pattern string, handler ...interface{}) *Trie {
	return r.Handle("OPTIONS", pattern, handler...)
}

// Head 为 HTTP HEAD request 设置路由
func (r Router) Head(pattern string, handler ...interface{}) *Trie {
	return r.Handle("HEAD", pattern, handler...)
}

// Any 为任意 HTTP method request 设置路由.
func (r Router) Any(pattern string, handler ...interface{}) *Trie {
	return r.Handle("any", pattern, handler...)
}

// Root 返回 method 对应的 *Trie 根节点.
func (r Router) Root(method string) *Trie {
	return r[method]
}

// Handle 为 HTTP method request 设置路由的通用形式.
// 参数 method 为 "*" 等效 "any". 其它值不做处理, 直接和 http.Request.Method 比较.
func (r Router) Handle(method string, pattern string, handler ...interface{}) *Trie {
	if method == "*" {
		method = "any"
	}

	t := r[method]
	if t == nil {
		t = newTrie('/')
		r[method] = t
	}

	trie := t.Mix(pattern)

	switch len(handler) {
	case 0:
	case 1:
		trie.Word = handler[0]
	default:
		trie.Word = handler
	}

	return trie
}

// HostRouter 是个简单的 Host 路由
type HostRouter struct {
	host        *Trie
	HandleError func(error, http.ResponseWriter, *http.Request) // 处理路由匹配错误
}

// NewHostRouter
func NewHostRouter() *HostRouter {
	return &HostRouter{newTrie('.'), HandleError}
}

// Add 添加 host 路由 handler.
func (r *HostRouter) Add(pattern string, handler ...interface{}) *Trie {
	if strings.IndexByte(pattern, '/') != -1 {
		panic("rivet: invalid host pattern: " + pattern)
	}

	t := r.host.Add(pattern)
	t.Word = ToDispatcher(handler...)
	return t
}

// ServeHTTP
func (r *HostRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	trie, params, err := r.host.Match(req.Host, req)

	if err != nil {
		r.HandleError(err, rw, req)
		return
	}

	if trie == nil {
		r.HandleError(StatusNotFound, rw, req)
		return
	}
	d, ok := trie.Word.(Dispatcher)

	if !ok {
		r.HandleError(StatusNotImplemented, rw, req)
		return
	}

	if d.IsInjector() {
		d.Dispatch(&Context{Params: params, Res: rw, Req: req})
	} else {
		d.Hand(params, rw, req)
	}
}
