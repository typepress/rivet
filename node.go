package rivet

/**
node 负责通过 Context 调用 Handler, 处理 http Request.
*/
type node struct {
	id      int
	riveter Riveter
	handler []Handler
}

// NewNode 返回内建的 Node 实例.
func NewNode(id int) Node {
	return &node{id: id}
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

	if n.riveter != nil {
		c = n.riveter(c.Response(), c.Request(), c.PathParams())
	}
	c.Handlers(n.handler...)
	c.Next()
}
