package rivet

import (
	"net/http"
)

/**
Router 管理路由, 通过匹配到的 Node 调用 Context.Next.
*/
type Router struct {
	rivet   Riveter
	trees   map[string]*Trie
	nodes   []Node
	newNode NodeBuilder
}

/**
NewRouter 新建一个 Router, 并设置 NotFound 为 http.NotFound.
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
func (r *Router) Get(pattern string, h ...Handler) Node {
	return r.add("GET", pattern, h)
}

// Post 为 HTTP POST request 设置路由
func (r *Router) Post(pattern string, h ...Handler) Node {
	return r.add("POST", pattern, h)
}

// Put 为 HTTP PUT request 设置路由
func (r *Router) Put(pattern string, h ...Handler) Node {
	return r.add("PUT", pattern, h)
}

// Patch 为 HTTP PATCH request 设置路由
func (r *Router) Patch(pattern string, h ...Handler) Node {
	return r.add("PATCH", pattern, h)
}

// Delete 为 HTTP DELETE request 设置路由
func (r *Router) Delete(pattern string, h ...Handler) Node {
	return r.add("DELETE", pattern, h)
}

// Options 为 HTTP OPTIONS request 设置路由
func (r *Router) Options(pattern string, h ...Handler) Node {
	return r.add("OPTIONS", pattern, h)
}

// Head 为 HTTP HEAD request 设置路由
func (r *Router) Head(pattern string, h ...Handler) Node {
	return r.add("HEAD", pattern, h)
}

// Any 为任意 HTTP method request 设置路由.
func (r *Router) Any(pattern string, h ...Handler) Node {
	return r.add("*", pattern, h)
}

/**
Handle 为 HTTP method request 设置路由的通用形式.
如果 method, pattern 对应的路由重复, 直接返回对应的节点. 否则添加新节点.
参数:
	method  "*" 等效 Any. 其它值不做处理, 直接和 http.Request.Method 比较.
	pattern 为空等效 NotFound 方法.

事实上 Router 不限制 method 的名称, 可随意定义.
*/
func (r *Router) Handle(method string, pattern string, h ...Handler) Node {
	return r.add(method, pattern, h)
}

// NotFound 设置匹配失败路由, 此路由只有一个. Node.Id() 固定为 0.
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
如果匹配失败, 返回 NotFound 节点.
*/
func (r *Router) Match(method, urlPath string) (Params, Node) {

	params, trie := r.trees[method].Match(urlPath)

	if trie == nil && method == "HEAD" {
		params, trie = r.trees["GET"].Match(urlPath)
	}

	if trie == nil && method != "*" {
		params, trie = r.trees["*"].Match(urlPath)
	}

	if trie == nil || trie.id == 0 {
		return nil, r.nodes[0]
	}

	return params, r.nodes[trie.id]
}

/**
RootTrie 返回 method 对应的 *Trie 根节点.
*/
func (r *Router) RootTrie(method string) *Trie {
	return r.trees[method]
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
	if trie.id != 0 {
		r.nodes[trie.id].Handlers(handlers...)
		return r.nodes[trie.id]
	}

	trie.id = len(r.nodes)

	_, keys, _ := parsePattern(pattern)
	node := r.newNode(trie.id, keys...)
	node.Handlers(handlers...)
	r.nodes = append(r.nodes, node)

	return node
}
