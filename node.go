package rivet

/**
node 负责通过 Context 调用 Handler, 处理 http Request.
*/
type node struct {
	id      int
	keys    map[string]bool
	riveter Riveter
	handler []Handler
}

/**
NewNode 返回内建的 Node 实例.
参数:
	id  识别号码
	key 用于过滤 URL.Path 参数名, 缺省全通过
*/
func NewNode(id int, key ...string) Node {
	n := new(node)
	n.id = id

	if len(key) != 0 {

		n.keys = make(map[string]bool, len(key))
		for _, k := range key {
			n.keys[k] = true
		}
	}

	return n
}

func (n *node) Id() int {
	return n.id
}

func (n *node) Riveter(riveter Riveter) {
	n.riveter = riveter
}

func (n *node) Handlers(handler ...Handler) {
	n.handler = handler
}

func (n *node) Apply(c Context) {
	if n == nil {

		if c == nil {
			panic("rivet: internal error, *base is nil")
		}

		req := c.Request()
		panic("rivet: internal error, *base is nil for " +
			req.Method + " \"" + req.Host + req.URL.Path + "\"")
	}

	params := c.Params()

	if n.keys != nil && len(params) != len(n.keys) {
		for k, _ := range params {
			if !n.keys[k] {
				delete(params, k)
			}
		}
	}

	if n.riveter != nil {
		c = n.riveter(c.Response(), c.Request(), params)
	}
	c.Handlers(n.handler...)
	c.Next()
}
