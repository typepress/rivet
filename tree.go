package rivet

import (
	"fmt"
)

/**
前缀树
NewRootRoute 已经已经设置好根节点, 因此节点总是已经设置好.
三种节点:
	定值节点     pattern==nil && indices!=nil
	模式节点     pattern!=nil
	模式分组节点 pattern==nil && indices==nil

举例:	 "/:name" 和 "/:cat" 是被允许的, 生成的解构
		定值节点  "/"
		模式分组      ""
		模式节点          ":name"
		模式节点          ":cat"

	定值节点的子节点最多只能包含一个模式节点或者模式分组节点, 索引用 0x0.
	模式分组的 path 为 "", 子节点为模式节点数组.
*/
type Trie struct {
	*base
	pattern *pattern
	nodes   []*Trie
	indices []byte
	path    string
}

/**
NewRootRoute 返回新的路由根节点, 已经设置路径为 "/".
*/
func NewRootRoute() *Trie {
	return &Trie{path: "/", indices: []byte{}}
}

/**
匹配 path, 返回匹配到的节点, 和提取到的参数.
*/
func (t *Trie) Match(path string) (Params, *Trie) {
	params, leaf := t.match(path)
	if leaf == nil {
		return nil, nil
	}

	return params, leaf
}
func (t *Trie) match(path string) (Params, *Trie) {
	var (
		i, j     int
		c, idx   byte
		child    *Trie
		catchAll *Trie
		all      string
		params   Params
	)

	if t == nil || len(path) == 0 {
		return nil, nil
	}

WALK:
	for {
		if len(path) == 0 {
			break
		}

		if len(t.path) == 0 {
			// 模式分组

			if params == nil {
				params = Params{}
			}

			// path 分段
			for i = 0; i < len(path); i++ {
				if path[i] == '/' {
					break
				}
			}

			// 保存 catchAll 避免回溯
			if t.nodes[0].path == "**" {
				catchAll = t.nodes[0]
				all = path
			}

			for j = len(t.nodes) - 1; j >= 0; j-- {
				child = t.nodes[j]

				if j == 0 && child.path == "**" {
					continue
				}

				if child.pattern.Match(path[:i], params) {
					t = child
					path = path[i:]
					break
				}
			}

			if j == -1 {
				break
			}

		} else if t.pattern == nil {
			// 定值节点

			if len(t.path) > len(path) {
				break
			}

			if len(t.path) == len(path) {
				if t.path == path {
					path = ""
				}
				break
			}

			if t.path != path[:len(t.path)] {
				break
			}

			path = path[len(t.path):]

		} else {
			// 模式节点

			if params == nil {
				params = Params{}
			}

			if t.path == "**" {
				params["*"] = path
				return params, t
			}

			// path 分段
			for i = 0; i < len(path); i++ {
				if path[i] == '/' {
					break
				}
			}

			if !t.pattern.Match(path[:i], params) {
				break
			}

			path = path[i:]
		}

		if len(path) == 0 {
			break
		}

		// 子节点, 按照索引匹配
		c = path[0]
		for i, idx = range t.indices {
			if c == idx {
				t = t.nodes[i]
				continue WALK
			}
		}

		// 失败, 必须含有模式分组, 下标和索引都是 0
		if len(t.indices) == 0 || t.indices[0] != 0 {
			break
		}

		t = t.nodes[0]
	}

	if len(path) == 0 {

		if t.base != nil {
			return params, t
		}

		if len(t.indices) != 0 && t.indices[0] == 0 {
			// catch-all
			if t.nodes[0].path == "" {
				t = t.nodes[0].nodes[0]
			} else {
				t = t.nodes[0]
			}

			if t.base != nil && t.path == "**" {
				if params == nil {
					params = Params{}
				}
				params["*"] = ""
				return params, t
			}
		}
	}

	if catchAll == nil {
		return nil, nil
	}

	params["*"] = all
	return params, catchAll
}

/**
Add 解析 path 增加节点.
返回值是叶子节点, 此节点可能会被后续根节点增加的节点覆盖.
所以保存此节点路由要使用 t.Route().
*/
func (t *Trie) Add(path string) *Trie {
	if len(path) == 0 {
		return nil
	}

	t = t.add(path)
	if t.base == nil {
		t.base = new(base)
	}
	return t
}

/**
Route 返回 Trie 节点中的路由.
注意 Trie 和路由的关系. Trie 的节点会发生改变.
此方法返回的 Route 保持不变.
*/
func (t *Trie) Route() Route {
	return t.base
}

func (t *Trie) add(path string) *Trie {

	var i, j int
	var child *Trie

	for {

		j = len(path)
		if j == 0 {
			if len(t.path) == 0 {
				panic("rivet: internal error, add a pattern group?")
			}
			return t
		}

		if len(t.path) < len(path) {
			j = len(t.path)
		}

		// 模式分组, 子节点枚举匹配
		if j == 0 {
			if path[0] != ':' && path[0] != '*' {
				panic("rivet: internal error form pattern group for: " + path)
			}

			// 提取模式段
			for i = 0; i < len(path); i++ {
				if path[i] == '/' {
					break
				}
			}

			// 是否有重复
			for j = 0; j < len(t.nodes); j++ {
				if t.nodes[j].path == path[:i] {
					break
				}
			}

			if j < len(t.nodes) {
				// 重复
				t = t.nodes[j]
				path = path[i:]
				continue
			}

			// 新增

			child = new(Trie)
			child.path = path[:i]
			child.pattern = newPattern(child.path)
			path = path[i:]

			if child.path == "**" && len(path) != 0 {
				panic("rivet: catch-all routes are only allowed at the end of the path")
			}

			if child.path == "**" {
				// 保持 "**" 位于第一个
				t.nodes = append([]*Trie{child}, t.nodes...)
			} else {
				t.nodes = append(t.nodes, child)
			}
			t = child
			continue
		}

		// 找出首个不同字节的下标
		for i = 0; i < j; i++ {

			if t.path[i] != path[i] {
				break
			}
		}

		// 模式节点, 分割为模式分组
		if len(t.path) != i && t.pattern != nil && (path[0] == ':' || path[0] == '*') {
			// copy 到新节点
			child = new(Trie)
			child.base = t.base
			child.pattern = t.pattern
			child.path = t.path
			child.indices = t.indices
			child.nodes = t.nodes

			t.base = nil
			t.pattern = nil
			t.path = ""
			t.indices = nil
			t.nodes = []*Trie{child}
			continue
		}

		// ==================== 添加子节点 =======================

		// 去掉 t.path 和 path 相同前缀部分
		path = path[i:]

		/**
		t.path 和 path 有相同前缀, 需要分割出新节点.
		i == 0, 一定是 模式节点
		*/
		if i != 0 && len(t.path) > i {
			child = new(Trie)
			child.base = t.base
			child.pattern = t.pattern
			child.path = t.path[i:]
			child.nodes = t.nodes
			child.indices = t.indices

			t.base = nil
			t.pattern = nil
			t.path = t.path[:i]
			t.nodes = []*Trie{child}
			t.indices = []byte{child.path[0]}
		}

		if len(path) == 0 {
			return t
		}

		// 查找 ":","*"
		for i = 0; i < len(path); i++ {
			if path[i] == ':' || path[i] == '*' {
				break
			}
		}

		// 定值子节点
		if i != 0 {
			for j = 0; j < len(t.indices); j++ {
				if t.indices[j] == path[0] {
					break
				}
			}
			// 匹配子节点
			if j < len(t.indices) {
				t = t.nodes[j]
				continue
			}

			child = new(Trie)
			child.path = path[:i]
			child.indices = []byte{} // 不能省略, 判断依据

			t.indices = append(t.indices, path[0])
			t.nodes = append(t.nodes, child)

			t = child
			path = path[i:]
			continue
		}

		// 已经是模式节点或者模式分组
		if len(t.indices) != 0 && t.indices[0] == 0 {
			t = t.nodes[0]
			continue
		}
		// 模式子节点
		for ; i < len(path); i++ {
			if path[i] == '/' {
				break
			}
		}

		child = new(Trie)
		child.path = path[:i]
		child.pattern = newPattern(child.path)
		path = path[i:]
		if child.path == "**" && len(path) != 0 {
			panic("rivet: catch-all routes are only allowed at the end of the path")
		}

		// 模式子节点索引为 0x0, 只能有一个, 位于 nodes[0]
		t.indices = append([]byte{0}, t.indices...)
		t.nodes = append([]*Trie{child}, t.nodes...)
		t = child
	}
}

/**
Print 用于调试输出, 便于查看 Trie 的结构.
参数:
	prefix 行前缀

输出格式:
[RPG] 缩进'path' 子节点数量[子节点首字符]

	R 表示路由
	P 表示模式节点
	G 表示模式分组
*/
func (t *Trie) Print(prefix string) {
	info := []byte{' ', ' ', ' '}
	if t.base != nil {
		info[0] = 'R'
	}
	if t.pattern != nil {
		info[1] = 'P'
	}
	if len(t.path) == 0 {
		info[2] = 'G'
	}

	fmt.Printf("[%v] %s'%s' %4d[%s]\n", string(info), prefix, t.path, len(t.nodes), string(t.indices))

	for l := len(t.path); l >= 0; l-- {
		prefix += " "
	}
	for _, child := range t.nodes {
		child.Print(prefix)
	}
}
