package rivet

import (
	"net/http"
)

type routes interface {
	Add(method, pattern string) Route
	Match(urls []string) (Params, Route)
}

type router struct {
	rivet     Riveter
	notFounds *base
	trees     map[string]*Trie
}

/**
Newrouter 创建符合 router 接口的实例.
参数:
	rivet 用于生成 Context 实例, 如果为 nil 使用 New() 创建一个.
*/
func NewRouter(rivet Riveter) Router {
	if rivet == nil {
		rivet = New()
	}

	return &router{
		rivet: rivet,
		trees: map[string]*Trie{},
	}
}

func (r *router) Get(pattern string, h ...Handler) Route {
	return r.add("GET", pattern, h)
}

func (r *router) Post(pattern string, h ...Handler) Route {
	return r.add("POST", pattern, h)
}

func (r *router) Put(pattern string, h ...Handler) Route {
	return r.add("PUT", pattern, h)
}

func (r *router) Patch(pattern string, h ...Handler) Route {
	return r.add("PATCH", pattern, h)
}

func (r *router) Delete(pattern string, h ...Handler) Route {
	return r.add("DELETE", pattern, h)
}

func (r *router) Options(pattern string, h ...Handler) Route {
	return r.add("OPTIONS", pattern, h)
}

func (r *router) Head(pattern string, h ...Handler) Route {
	return r.add("HEAD", pattern, h)
}

func (r *router) Any(pattern string, h ...Handler) Route {
	return r.add("*", pattern, h)
}

func (r *router) Add(method string, pattern string, h ...Handler) Route {
	return r.add(method, pattern, h)
}

func (r *router) NotFound(h ...Handler) Route {
	route := &base{handlers: h}
	if h == nil {
		r.notFounds = nil
	} else {
		r.notFounds = route
	}
	return route
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	method := req.Method
	urlPath := req.URL.Path

	params, route := r.trees[method].Match(urlPath)

	if route == nil && method == "HEAD" {
		params, route = r.trees["GET"].Match(urlPath)
	}

	if route == nil && method != "*" {
		params, route = r.trees["*"].Match(urlPath)
	}

	if route != nil {
		route.Apply(params, r.rivet.Context(res, req))
		return
	}

	if r.notFounds != nil {
		r.notFounds.Apply(nil, r.rivet.Context(res, req))
		return
	}

	http.NotFound(res, req)
}

func (r *router) Match(method, urlPath string) (Params, Route) {

	params, route := r.trees[method].Match(urlPath)

	if route == nil && method == "HEAD" {
		params, route = r.trees["GET"].Match(urlPath)
	}

	if route == nil && method != "*" {
		params, route = r.trees["*"].Match(urlPath)
	}

	if route == nil {
		return nil, nil
	}

	return params, route
}

func (r *router) add(method string, pattern string, handlers []Handler) Route {
	var route Route

	if pattern == "" {
		return r.NotFound(handlers...)
	}
	if pattern[0] != '/' {
		panic(`rivet: invalide pattern`)
	}

	t := r.trees[method]
	if t == nil {
		t = NewRoot()
		r.trees[method] = t
	}
	route = t.Add(pattern)

	route.Handlers(handlers...)
	return route
}
