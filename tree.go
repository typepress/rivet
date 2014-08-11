package rivet

import (
	"fmt"
	"sort"
)

func slashCount(path string) (c int) {
	size := len(path)
	for i := 0; i < size; i++ {
		switch path[i] {
		case '/':
			c++
		}
	}
	return
}

func parsePattern(path string) (slash int, keys []string, cathAll bool) {

	size := len(path)
	keys = []string{}

	for i := 0; i < size; i++ {
		switch path[i] {
		case '/':
			slash++
		case ':', '*':
			if i+1 < size && path[i] == path[i+1] {
				cathAll = true
				if i+2 != size {
					panic("rivet: catch-all must be end of pattern. " + path)
				}
				keys = append(keys, "*")
				break
			}

			j := i + 1
			k := 0
			for ; i < size; i++ {
				if k == 0 && path[i] == ' ' {
					k = i
				}
				if path[i] == '/' {
					slash++
					break
				}
			}

			if k == 0 {
				k = i
			}

			if path[j:k] != "" {
				keys = append(keys, path[j:k])
			}
		}
	}
	return
}

/**
Trie 不直接管理路由, 通过用户可设置字段 Id 组织管理 URL path.
用户通过字段 Id 自己维护路由, 0 值表示非路由节点.
NewRootTrie 已经已经设置好根节点, 因此节点总是已经设置好.
三种节点:
	定值节点     pattern==nil && indices!=nil
	模式节点     pattern!=nil
	模式分组节点 pattern==nil && indices==nil && path==""

举例:	 "/:name" 和 "/:cat" 是被允许的, 生成的解构
		定值节点  "/"
		模式分组      ""
		模式节点          ":name"
		模式节点          ":cat"

	定值节点的子节点最多只能包含一个模式节点或者模式分组节点, 索引用 0x0.
	模式分组的 path 为 "", 子节点为模式节点数组.
*/
type Trie struct {
	*perk
	Id       int // 用户数据标识, 0 表示非路由节点
	nodes    []*Trie
	indices  []byte
	path     string
	slash    int // path 中的斜线个数
	slashMax int // 后续 tree 中的斜线最大个数
}

/**
NewRootTrie 返回新的根节点 Trie, 已经设置路径为 "/".
*/
func NewRootTrie() *Trie {
	return &Trie{path: "/", indices: []byte{}, slash: 1, slashMax: 1}
}

/**
Match 匹配 url path, 返回匹配到的节点, 和提取到的参数.
*/
func (t *Trie) Match(path string) (Params, *Trie) {

	var (
		i, j     int
		slashMax int
		c, idx   byte
		catchAll *Trie
		all      string
		params   Params
	)

	if t == nil || len(path) == 0 {
		return nil, nil
	}

	// 默认从定值节点匹配
	if len(t.path) > len(path) {
		return nil, nil
	}

	if t.path != path[:len(t.path)] {
		return nil, nil
	}

	path = path[len(t.path):]

	// 匹配子节点
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
			if t.nodes[0].name == "*" {
				catchAll = t.nodes[0]
				all = path
			}

			if slashMax <= 0 {
				slashMax = slashCount(path[i:])
			}

			for j = len(t.nodes) - 1; j >= 0; j-- {

				if j != 0 && t.nodes[j].slashMax < slashMax {
					continue
				}

				if t.nodes[j].Perk(path[:i], params) {
					t = t.nodes[j]
					path = path[i:]
					break
				}
			}

			if j < 0 || len(path) == 0 {
				break
			}
			if len(t.path) == 0 {
				continue
			}
		}

		// 子节点, 按照索引匹配, 能匹配上的一定是定值节点
		c = path[0]
		for i, idx = range t.indices {
			if c == idx {
				if len(t.nodes[i].path) <= len(path) &&
					t.nodes[i].path == path[:len(t.nodes[i].path)] {

					t = t.nodes[i]
					path = path[len(t.path):]
					continue WALK
				}
				break
			}
		}

		// 失败, 必须含有模式分组, 下标和索引都是 0
		if len(t.indices) == 0 || t.indices[0] != 0 {
			break
		}

		t = t.nodes[0]

		// 分组继续
		if len(t.path) == 0 {
			continue
		}

		// 模式节点
		if params == nil {
			params = Params{}
		}

		if t.name == "*" {
			params["*"] = path
			return params, t
		}

		// path 分段
		for i = 0; i < len(path); i++ {
			if path[i] == '/' {
				break
			}
		}

		if !t.Perk(path[:i], params) {
			break
		}
		path = path[i:]
	}

	if len(path) == 0 {

		if t.Id != 0 {
			return params, t
		}

		if len(t.indices) != 0 && t.indices[0] == 0 {
			// catch-all
			if t.nodes[0].path == "" {
				t = t.nodes[0].nodes[0]
			} else {
				t = t.nodes[0]
			}

			if t.Id != 0 && t.name == "*" {
				if params == nil {
					params = Params{}
				}
				params["*"] = ""
				return params, t
			}
		}
	}

	if catchAll == nil || catchAll.Id == 0 {
		return nil, nil
	}
	params["*"] = all
	return params, catchAll
}

/**
Add 解析 path, 确定叶子节点
返回值:

	*Trie 对应的叶子节点, 如果 path 重复, 返回原有节点.
		此节点可能会被后续根节点增加的节点覆盖, 因此应该及时设置 Id.
		这意味着 Add 方法非并发安全.
		调用者应该先判断字段 Id 是否为 0, 确定数据关系.

缺陷:
	此方法暂时只支持 Trie 为根节点.
*/
func (t *Trie) Add(path string) *Trie {
	var i, j int
	var child *Trie

	if t.path != "/" || len(path) == 0 || path[0] != '/' {
		panic("rivet: Add supported only from root Trie.")
	}

	slashMax := slashCount(path)

	for {
		j = len(path)

		if j == 0 {
			if len(t.path) == 0 {
				panic("rivet: internal error, add a pattern group?")
			}
			return t
		}

		if t.perk != nil && t.name == "*" {
			panic("rivet: catch-all routes are only allowed at the end of the path")
		}

		if len(t.path) < len(path) {
			j = len(t.path)
		}

		if t.slashMax < slashMax {
			t.slashMax = slashMax
		}
		slashMax -= t.slash

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

			// 重复
			if j < len(t.nodes) {
				t = t.nodes[j]
				path = path[i:]
				continue
			}

			// 新增

			child = new(Trie)
			child.path = path[:i]
			child.perk = newPerk(child.path)
			path = path[i:]

			if child.name == "*" {
				// 保持 "**" 位于第一个
				t.nodes = append([]*Trie{child}, t.nodes...)
			} else {

				i = sort.Search(len(t.nodes), func(i int) bool {
					if t.nodes[i].name == "*" {
						return false
					}
					return t.nodes[i].slashMax < slashMax
				})

				t.nodes = append(t.nodes, nil)
				for j = len(t.nodes) - 1; j > i; j-- {
					t.nodes[j] = t.nodes[j-1]
				}
				t.nodes[i] = child
			}
			t = child
			continue
		}

		// ==================== 添加子节点 =======================
		// 找出首个不同字节的下标
		for i = 0; i < j; i++ {

			if t.path[i] != path[i] {
				break
			}
		}

		// 去掉 t.path 和 path 相同前缀部分
		path = path[i:]

		/**
		t.path 和 path 有相同前缀, 需要分割出新节点.
		i != 0, t 一定是 定值节点, 模式节点不会产生分割
		*/
		if i != 0 && len(t.path) > i {
			child = new(Trie)
			child.Id = t.Id
			child.perk = t.perk
			child.path = t.path[i:]
			child.nodes = t.nodes
			child.indices = t.indices

			t.Id = 0
			t.perk = nil
			t.path = t.path[:i]
			t.nodes = []*Trie{child}
			t.indices = []byte{child.path[0]}

			child.slashMax = t.slashMax - t.slash
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

		// 新子节点是 定值节点
		if i != 0 {
			for j = 0; j < len(t.indices); j++ {
				if t.indices[j] == path[0] {
					break
				}
			}

			// 重复子节点
			if j < len(t.indices) {
				t = t.nodes[j]
				continue
			}

			// 增加子节点
			child = new(Trie)
			child.path = path[:i]
			child.indices = []byte{} // 不能省略, 判断依据
			child.slashMax = t.slashMax - slashCount(t.path)
			child.slash = slashCount(child.path)

			path = path[i:]

			i = sort.Search(len(t.nodes), func(i int) bool {
				if t.indices[i] == 0 {
					return false
				}
				return t.nodes[i].slashMax < child.slashMax
			})

			t.nodes = append(t.nodes, nil)
			t.indices = append(t.indices, 0)
			for j = len(t.nodes) - 1; j > i; j-- {
				t.nodes[j] = t.nodes[j-1]
				t.indices[j] = t.indices[j-1]
			}

			t.nodes[i] = child
			t.indices[i] = child.path[0]

			t = child

			continue
		}

		// 新子节点是模式节点

		// t 的子节点已有模式节点或分组
		if len(t.indices) != 0 && t.indices[0] == 0 {
			t = t.nodes[0]
			// 分组节点, 继续循环
			if len(t.path) == 0 {
				continue
			}
			// 分割为分组节点
			child = new(Trie)
			child.Id = t.Id
			child.perk = t.perk
			child.path = t.path
			child.indices = t.indices
			child.nodes = t.nodes
			child.slash = t.slash
			child.slashMax = t.slashMax

			t.Id = 0
			t.perk = nil
			t.path = ""
			t.indices = nil
			t.nodes = []*Trie{child}
			t.slash = 0
			continue
		}

		// t 的子节点没有模式节点或分组
		for ; i < len(path); i++ {
			if path[i] == '/' {
				break
			}
		}

		child = new(Trie)
		child.path = path[:i]
		child.perk = newPerk(child.path)
		child.slash = slashCount(child.path)
		path = path[i:]

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
Id 斜线个数[RPG] 缩进'path' 子节点数量[子节点首字符]

	R 表示路由
	P 表示模式节点
	G 表示模式分组
*/
func (t *Trie) Print(prefix string) {
	info := []byte{' ', ' ', ' '}

	if t.Id != 0 {
		info[0] = 'R'
	}
	if t.perk != nil {
		info[1] = 'P'
	}
	if len(t.path) == 0 {
		info[2] = 'G'
	}

	fmt.Printf("%4d %2d[%v] %s'%s' %4d[%s]\n", t.Id, t.slashMax, string(info), prefix, t.path, len(t.nodes), string(t.indices))

	for l := len(t.path); l >= 0; l-- {
		prefix += " "
	}
	for _, child := range t.nodes {
		child.Print(prefix)
	}
}
