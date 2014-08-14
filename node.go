package rivet

/**
node 负责通过 Context 调用 Handler, 处理 http Request.
*/
type node struct {
	id      int
	keys    map[string]bool
	riveter Riveter
	handler []interface{}
}

/**
NewNode 返回内建的 Node 实例.
参数:
	id  识别号码
	key 用于过滤 URL.Path 参数名, 缺省全通过.
		内建 Node 实现中, 如果设置了 key, Apply 方法会删除
		context.Params() 中 key 之外的数据. 算法: 如果

			 key != nil && len(key) != len(context.Params())

			为真, 删除多余的数据.
*/
func NewNode(id int, key ...string) Node {
	n := new(node)
	n.id = id

	if len(key) != 0 {

		n.keys = make(map[string]bool)
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

func (n *node) Handlers(handler ...interface{}) {
	n.handler = handler
}

func (n *node) Apply(c Context) {

	var params Params

	if n == nil {

		if c == nil {
			panic("rivet: internal error, *base is nil")
		}

		req := c.Request()
		panic("rivet: internal error, *base is nil for " +
			req.Method + " \"" + req.Host + req.URL.Path + "\"")
	}

	params = c.Params()

	if n.keys != nil && len(params) != len(n.keys) {

		clear := len(n.keys) == 0
		for k, _ := range params {

			if clear || !n.keys[k] {
				delete(params, k)
			}
		}
	}

	if n.riveter != nil {
		c = n.riveter(c.Response(), c.Request())
	}
	c.Handlers(n.handler...)
	c.Next()
}
