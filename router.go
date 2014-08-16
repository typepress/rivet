package rivet

import (
	"net/http"
)

/**
Router 管理路由.
*/
type Router struct {
	rivet   Riveter
	trees   map[string]*Trie
	nodes   []Node
	newNode NodeBuilder
}

/**
NewRouter 新建 *Router, 并设置 NotFound Handler 为 http.NotFound.
参数:
	rivet 用于生成 Context 实例, 如果为 nil 使用 NewContext 创建.
*/
func NewRouter(rivet Riveter) *Router {
	if rivet == nil {
		rivet = NewContext
	}

	notFound := NewNode(0)
	notFound.Handlers(http.NotFound)

	return &Router{
		newNode: NewNode,
		rivet:   rivet,
		trees:   map[string]*Trie{},
		nodes:   []Node{notFound},
	}
}

/**
设置 NodeBuilder, 默认使用 NewNode.
*/
func (r *Router) NodeBuilder(nb NodeBuilder) {
	if nb != nil {
		r.newNode = nb
	}
}

// Get 为 HTTP GET request 设置路由
func (r *Router) Get(pattern string, handler ...interface{}) Node {
	return r.add("GET", pattern, handler)
}

// Post 为 HTTP POST request 设置路由
func (r *Router) Post(pattern string, handler ...interface{}) Node {
	return r.add("POST", pattern, handler)
}

// Put 为 HTTP PUT request 设置路由
func (r *Router) Put(pattern string, handler ...interface{}) Node {
	return r.add("PUT", pattern, handler)
}

// Patch 为 HTTP PATCH request 设置路由
func (r *Router) Patch(pattern string, handler ...interface{}) Node {
	return r.add("PATCH", pattern, handler)
}

// Delete 为 HTTP DELETE request 设置路由
func (r *Router) Delete(pattern string, handler ...interface{}) Node {
	return r.add("DELETE", pattern, handler)
}

// Options 为 HTTP OPTIONS request 设置路由
func (r *Router) Options(pattern string, handler ...interface{}) Node {
	return r.add("OPTIONS", pattern, handler)
}

// Head 为 HTTP HEAD request 设置路由
func (r *Router) Head(pattern string, handler ...interface{}) Node {
	return r.add("HEAD", pattern, handler)
}

// Any 为任意 HTTP method request 设置路由.
func (r *Router) Any(pattern string, handler ...interface{}) Node {
	return r.add("*", pattern, handler)
}

/**
Handle 为 HTTP method request 设置路由的通用形式.
如果 method, pattern 对应的路由重复, 直接返回对应的节点. 否则添加新节点.
参数:
	method  "*" 等效 Any. 其它值不做处理, 直接和 http.Request.Method 比较.
	pattern 为空等效 NotFound 方法.

事实上 Router 不限制 method 的名称, 可随意定义.
*/
func (r *Router) Handle(method string, pattern string, h ...interface{}) Node {
	return r.add(method, pattern, h)
}

// NotFound 设置匹配失败路由, 此路由只有一个. Node.Id() 固定为 0.
func (r *Router) NotFound(h ...interface{}) Node {
	return r.add("", "", h)
}

// http.Handler
func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	method := req.Method
	urlPath := req.URL.Path

	c := r.rivet(rw, req)
	rw = c.Response()
	trie := r.trees[method].Match(urlPath, c, rw, req)

	if trie == nil && method == "HEAD" {
		trie = r.trees["GET"].Match(urlPath, c, rw, req)
	}

	if trie == nil && method != "*" {
		trie = r.trees["*"].Match(urlPath, c, rw, req)
	}

	if trie == nil {
		r.nodes[0].Apply(c)
	} else {
		r.nodes[trie.id].Apply(c)
	}
}

/**
Match 匹配路由节点. 如果匹配失败, 返回 NotFound 节点.
参数:
	method   Request.Method, 确定对应的 Root Trie.
	urlPath  Request.URL.Path, 传递给 Trie.
	rec      URL.Path 参数接收器, 传递给 Trie.
	rw       http 响应, 传递给 Trie.
	req      http 请求, 传递给 Trie.
*/
func (r *Router) Match(method, urlPath string, rec ParamsReceiver,
	rw http.ResponseWriter, req *http.Request) Node {

	trie := r.trees[method].Match(urlPath, rec, rw, req)

	if trie == nil && method == "HEAD" {
		trie = r.trees["GET"].Match(urlPath, rec, rw, req)
	}

	if trie == nil && method != "*" {
		trie = r.trees["*"].Match(urlPath, rec, rw, req)
	}

	if trie == nil {
		return r.nodes[0]
	}

	return r.nodes[trie.id]
}

/**
RootTrie 返回 method 对应的 *Trie 根节点.
*/
func (r *Router) RootTrie(method string) *Trie {
	return r.trees[method]
}

func (r *Router) add(method string, pattern string, handlers []interface{}) Node {

	if pattern == "" {
		r.nodes[0] = r.newNode(0)
		r.nodes[0].Handlers(handlers...)
		return r.nodes[0]
	}
	if pattern[0] != '/' {
		panic(`rivet: invalide pattern`)
	}

	t := r.trees[method]
	if t == nil {
		t = NewRootTrie()
		r.trees[method] = t
	}

	trie := t.Add(pattern, NewFilter)
	if trie.id != 0 {
		r.nodes[trie.id].Handlers(handlers...)
		return r.nodes[trie.id]
	}

	trie.id = len(r.nodes)

	node := r.newNode(trie.id)
	node.Handlers(handlers...)
	r.nodes = append(r.nodes, node)

	return node
}
