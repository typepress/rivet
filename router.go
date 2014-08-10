package rivet

import (
	"net/http"
)

/**
Router 通过匹配到的 Node 调用 Context.Invoke.
事实上为了能正常调用 Route.Match 方法, 生成 Router 的方法需要 Rivet 实例参数.
如果不设定 NotFound Handler, 直接调用 http.NotFound, 不调用 Context.Invoke.
*/
type Router struct {
	rivet   Riveter
	trees   map[string]*Trie
	nodes   []Node
	newNode func(int) Node
}

/**
NewRouter 新建一个路由管理器, 并设置 NotFounds.
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

// Get 为 HTTP GET request 添加路由
func (r *Router) Get(pattern string, h ...Handler) Node {
	return r.add("GET", pattern, h)
}

// Post 为 HTTP POST request 添加路由
func (r *Router) Post(pattern string, h ...Handler) Node {
	return r.add("POST", pattern, h)
}

// Put 为 HTTP PUT request 添加路由
func (r *Router) Put(pattern string, h ...Handler) Node {
	return r.add("PUT", pattern, h)
}

// Patch 为 HTTP PATCH request 添加路由
func (r *Router) Patch(pattern string, h ...Handler) Node {
	return r.add("PATCH", pattern, h)
}

// Delete 为 HTTP DELETE request 添加路由
func (r *Router) Delete(pattern string, h ...Handler) Node {
	return r.add("DELETE", pattern, h)
}

// Options 为 HTTP OPTIONS request 添加路由
func (r *Router) Options(pattern string, h ...Handler) Node {
	return r.add("OPTIONS", pattern, h)
}

// Head 为 HTTP HEAD request 添加路由
func (r *Router) Head(pattern string, h ...Handler) Node {
	return r.add("HEAD", pattern, h)
}

// Any 为任意 HTTP method request 添加路由.
func (r *Router) Any(pattern string, h ...Handler) Node {
	return r.add("*", pattern, h)
}

/**
Handle 为 HTTP method request 设置路由
参数:
	method  "*" 等效 Any. 其它值不做处理, 直接和 http.Request.Method 比较.
	pattern 为空等效 NotFound 方法.
*/
func (r *Router) Handle(method string, pattern string, h ...Handler) Node {
	return r.add(method, pattern, h)
}

// NotFound 设置匹配失败路由, 此路由只有一个.
func (r *Router) NotFound(h ...Handler) Node {
	return r.add("", "", h)
}

// http.Handler
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	params, node := r.Match(req.Method, req.URL.Path)
	node.Apply(r.rivet(res, req, params))
}

/**
Match 依据 method, 和 urlPath 匹配路由节点.
如果匹配失败, 返回 NotFounds 节点, 此节点 Id 为 0.
*/
func (r *Router) Match(method, urlPath string) (Params, Node) {

	params, trie := r.trees[method].Match(urlPath)

	if trie == nil && method == "HEAD" {
		params, trie = r.trees["GET"].Match(urlPath)
	}

	if trie == nil && method != "*" {
		params, trie = r.trees["*"].Match(urlPath)
	}

	if trie == nil || trie.Id == 0 {
		return nil, r.nodes[0]
	}

	return params, r.nodes[trie.Id]
}

func (r *Router) add(method string, pattern string, handlers []Handler) Node {

	if pattern == "" {
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

	trie := t.Add(pattern)

	if trie.Id != 0 {
		r.nodes[trie.Id].Handlers(handlers...)
		return r.nodes[trie.Id]
	}

	trie.Id = len(r.nodes)

	node := r.newNode(trie.Id)
	node.Handlers(handlers...)
	r.nodes = append(r.nodes, node)

	return node
}
