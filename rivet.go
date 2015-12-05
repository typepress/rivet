package rivet

import "net/http"

// Rivet 包装 Router, 实现了支持注入的 http.Handler.
type Rivet struct {
	router      Router
	HandleError func(error, http.ResponseWriter, *http.Request) // 处理路由匹配错误
}

// New 新建 *Rivet
func New() *Rivet {
	return &Rivet{
		router:      map[string]*Trie{},
		HandleError: HandleError,
	}
}

// ServeHTTP 实现了 http.Handler 接口.
func (r *Rivet) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	trie, params, err := r.router.Match(req.Method, req.URL.Path, req)

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

	if !d.IsInjector() {
		d.Handle(params, rw, req)
		return
	}

	d.Dispatch(NewContext(params, rw, req))
}

func (r *Rivet) Match(method, urlPath string, req *http.Request) (trie *Trie, params Params, err error) {
	return r.router.Match(method, urlPath, req)
}

func (r *Rivet) Get(pattern string, handler ...interface{}) *Trie {
	return r.Handle("GET", pattern, handler...)
}

func (r *Rivet) Post(pattern string, handler ...interface{}) *Trie {
	return r.Handle("POST", pattern, handler...)
}

func (r *Rivet) Put(pattern string, handler ...interface{}) *Trie {
	return r.Handle("PUT", pattern, handler...)
}

func (r *Rivet) Patch(pattern string, handler ...interface{}) *Trie {
	return r.Handle("PATCH", pattern, handler...)
}

func (r *Rivet) Delete(pattern string, handler ...interface{}) *Trie {
	return r.Handle("DELETE", pattern, handler...)
}

func (r *Rivet) Options(pattern string, handler ...interface{}) *Trie {
	return r.Handle("OPTIONS", pattern, handler...)
}

func (r *Rivet) Head(pattern string, handler ...interface{}) *Trie {
	return r.Handle("HEAD", pattern, handler...)
}

func (r *Rivet) Any(pattern string, handler ...interface{}) *Trie {
	return r.Handle("any", pattern, handler...)
}

func (r *Rivet) Root(method string) *Trie {
	return r.router[method]
}

// Handle 内部对 handler 进行了 Dispatcher 包装.
// 这意味着返回的 Trie.Word 为 nil 或者 Dispatcher.
func (r *Rivet) Handle(method string, pattern string, handler ...interface{}) *Trie {
	t := r.router.Handle(method, pattern)
	t.Word = ToDispatcher(handler...)
	return t
}
