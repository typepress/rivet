package rivet

import (
	"sort"
)

// 前缀树
type tree struct {
	*base
	*pattern
	fix   []*tree // 定值子节点
	rules []*tree // 模式子节点
	value string  // 原始 pattern
}

// 只为 root 生成
func newTree() *tree {
	return &tree{}
}

func (t *tree) Add(method, pattern string) Route {
	r := t.addNodes(pattern)
	if r != nil && r.base == nil {
		r.base = new(base)
	}
	return r
}

func (t *tree) Match(path string) (Params, *tree) {
	var (
		nodes []*tree
		n, i  int
	)

	if t == nil {
		return nil, nil
	}

	params := Params{}
	for ; i < len(path); i++ {

		nodes = t.fix
		n = sort.Search(len(nodes), func(i int) bool {

			if len(nodes[i].value) > len(path) {
				return nodes[i].value > path
			}
			return nodes[i].value > path[:len(nodes[i].value)]
		})
		n--

		if n != -1 &&
			len(nodes[n].value) <= len(path) &&
			nodes[n].value == path[:len(nodes[n].value)] {

			t = nodes[n]
			path = path[len(t.value):]
			i = -1
			continue
		}

		i++
		for ; i < len(path); i++ {
			if path[i] == '/' {
				break
			}
		}

		nodes = t.rules

		n = sort.Search(len(nodes), func(j int) bool {

			if len(nodes[j].prefix) > i {
				return nodes[j].prefix > path[:i]
			}

			return nodes[j].prefix > path[:len(nodes[j].prefix)]
		})
		n--

		t = nil
		// 贪心匹配
		for ; n >= 0 && len(nodes[n].prefix) <= i; n-- {

			if v, ok := nodes[n].pattern.Match(path[:i]); ok {
				if nodes[n].name != "" {
					params[nodes[n].name] = v
				}
				t = nodes[n]
				break
			}
		}

		if t == nil {
			return nil, nil
		}
		path = path[i:]
		i = -1
	}

	if len(path) == 0 {
		return params, t
	}
	return nil, nil
}

func (t *tree) addNodes(path string) *tree {
	var end int
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			end = i
			continue
		}
		if c != ':' && c != '*' {
			continue
		}

		if end != 0 {
			t = t.addFix(path[:end])
		}

		for ; i < len(path); i++ {
			if path[i] == '/' {
				break
			}
		}

		t = t.addRule(path[end:i]) //保留前缀
		path = path[i:]
		i = 0
		end = 0
	}

	return t.addFix(path)
}

// 模式节点
func (t *tree) addRule(value string) *tree {

	pattern := newPattern(value)

	nodes := t.rules
	eq := -1
	i := sort.Search(len(nodes), func(i int) bool {
		if nodes[i].value == value {
			eq = i
		}
		return nodes[i].prefix > pattern.prefix
	})
	if eq != -1 {
		return nodes[eq]
	}

	nodes = append(nodes, nil)
	for pos := len(nodes) - 1; pos > i; pos-- {
		nodes[pos] = nodes[pos-1]
	}

	child := new(tree)
	child.value = value
	child.pattern = pattern

	nodes[i] = child
	t.rules = nodes

	return child
}

// 定值节点
func (t *tree) addFix(value string) *tree {
	if value == "" {
		return t
	}

	nodes := t.fix
	i := sort.Search(len(nodes), func(i int) bool {
		return nodes[i].value >= value
	})

	if i < len(nodes) && nodes[i].value == value {
		return nodes[i]
	}

	nodes = append(nodes, nil)
	for pos := len(nodes) - 1; pos > i; pos-- {
		nodes[pos] = nodes[pos-1]
	}

	child := new(tree)
	child.value = value

	nodes[i] = child
	t.fix = nodes
	return child
}
