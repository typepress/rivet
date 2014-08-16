package rivet

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func analyzePath(path string) (c int, names map[string]bool) {

	size := len(path)
	for i := 0; i < size; i++ {
		if path[i] != ':' && path[i] != '*' {

			if path[i] == '/' {
				c++
			}

			continue
		}

		if i+1 < size && path[i] == path[i+1] {
			if names == nil {
				names = make(map[string]bool)
			}
			names["*"] = true
			break
		}

		j := i + 1
		k := 0

		for ; i < size; i++ {
			if path[i] == ' ' {
				k = i
				break
			}
			if path[i] == '/' {
				c++
				k = i
				break
			}
		}

		if k == 0 {
			k = i
		}

		if path[j:k] != "" {
			if names == nil {
				names = make(map[string]bool)
			}
			names[path[j:k]] = true
		}
	}
	return
}

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

// Trie 专用
type perk struct {
	filter Filter
	name   string // 空值匹配不提取
}

func newPerk(text string, newFilter FilterBuilder) *perk {
	if text[0] != ':' && text[0] != '*' {
		panic("rivet: internal error form newFilter : " + text)
	}

	a := strings.Split(text[1:], " ")

	p := new(perk)
	p.name = a[0]
	switch len(a) {
	case 1:
		// 优化处理, 无需建立 Filter

		// "/path/to/:pattern/to/**"
		if p.name == "*" || p.name == ":" {
			p.name = "*"
		}
	case 2:
		p.filter = newFilter(a[1])
	default:
		p.filter = newFilter(a[1], a[2:]...)
	}

	return p
}

func (p *perk) Filter(text string,
	rw http.ResponseWriter, req *http.Request) (interface{}, bool) {
	if p.filter == nil {
		return text, true
	}
	return p.filter.Filter(text, rw, req)
}

/**
discardParams 替代 ParamsReceiver 为 nil 的情况
*/
type discardParams bool

func (discardParams) ParamsReceiver(key, text string, val interface{}) {
}

var _discard = discardParams(true)

/**
Trie 管理路由 patterns.
Trie 不直接管理路由 Handler, 由使用者通过 SetId 进行组织管理.
id 为 0 的节点保留给内部算法使用, 所以 0 值表示非路由节点.

请使用 NewRootTrie 获得根节点.
使用 Print 方法有助于了 Trie 的结构和算法.
*/
type Trie struct {
	*perk
	path     string
	names    map[string]bool // 参数名
	nodes    []*Trie
	indices  []byte
	id       int  // 用户数据标识, 0 表示非路由节点
	slash    int  // path 中的斜线个数
	slashMax int  // 后续 tree 中的斜线最大个数
	catchAll bool // "/**"
	ots      bool // optional trailing slashes , 可选尾部斜线
}

/**
NewRootTrie 返回新的根节点 Trie, 已经设置路径为 "/".
*/
func NewRootTrie() *Trie {
	return &Trie{path: "/", indices: []byte{}, slash: 1, slashMax: 1}
}

/**
GetId 返回节点 id.
*/
func (t *Trie) GetId() int {
	if t == nil {
		return 0
	}
	return t.id
}

/**
SetId 设置节点 id. 设置条件为:

	id != 0 && t != nil && t.id == 0

其中:
	t.id == 0 为内部管理节点
*/
func (t *Trie) SetId(id int) {
	if id != 0 && t != nil && t.id == 0 {
		t.id = id
	}
}

/**
Match 匹配 URL.Path, 返回匹配到的节点.
参数:
	path 待匹配的 URL.Path
	rec  指定参数接收器, 如果为 nil 表示丢弃参数.
	rw, req 供 Filter 使用, 如果 Filter 不需要的话, 可以为 nil

返回:
	成功返回对应的节点, 该节点 GetId() 一定不为 0.
	失败返回 nil.
*/
func (t *Trie) Match(path string, rec ParamsReceiver,
	rw http.ResponseWriter, req *http.Request) *Trie {

	var (
		i, j     int
		slashMax int
		c, idx   byte
		catchAll *Trie
		all      string
		val      interface{}
		ok       bool
	)

	if t == nil || len(path) == 0 {
		return nil
	}

	// 默认从定值节点匹配
	if len(t.path) > len(path) {
		return nil
	}

	if t.path != path[:len(t.path)] {
		return nil
	}

	if rec == nil {
		rec = _discard
	}
	path = path[len(t.path):]

	// 匹配子节点
WALK:
	for {
		if len(path) == 0 {
			break
		}

		slashMax -= t.slash

		if len(t.path) == 0 {
			// 模式分组
			j = len(path)
			// path 分段
			for i = 0; i < j; i++ {
				if path[i] == '/' {
					break
				}
			}

			// 保存 catchAll 避免回溯
			if t.nodes[0].name == "*" {
				catchAll = t.nodes[0]
				all = path
			}

			if slashMax < 0 {
				slashMax = slashCount(path[i:])
			}

			for j = len(t.nodes) - 1; j >= 0; j-- {

				if j != 0 {
					if !t.nodes[j].catchAll &&
						t.nodes[j].slashMax != slashMax &&
						// 有可能 ots
						t.nodes[j].slashMax != slashMax+1 {
						continue
					}
				}

				if val, ok = t.nodes[j].Filter(path[:i], rw, req); ok {
					t = t.nodes[j]
					if t.name != "" {
						rec.ParamsReceiver(t.name, path[:i], val)
					}
					path = path[i:]
					break
				}
			}

			// 未匹配或者匹配完成
			if j < 0 || len(path) == 0 {
				break
			}
			// 匹配的仍然是模式分组
			if len(t.path) == 0 {
				continue
			}
			// 继续匹配
		}

		// 子节点, 按照索引匹配, 能匹配上的一定是定值节点
		c = path[0]
		for i, idx = range t.indices {
			if c == idx {
				if len(t.nodes[i].path) <= len(path) {

					if t.nodes[i].path == path[:len(t.nodes[i].path)] {

						c = 0
						path = path[len(t.nodes[i].path):]
					}
				} else if t.nodes[i].ots {
					// 未被分割的尾斜线

					if len(t.nodes[i].path) == len(path)+1 &&
						t.nodes[i].path[:len(path)] == path {

						c = 0
						path = ""
					}
				}
				break
			}
		}

		if c == 0 {

			// see Test_OTS
			if t.indices[0] == 0 {
				if t.nodes[0].catchAll {

					catchAll = t.nodes[0]
					if len(catchAll.path) == 0 &&
						catchAll.nodes[0].catchAll {
						catchAll = catchAll.nodes[0]
					}
				} else if len(t.nodes[0].path) == 0 &&
					t.nodes[0].nodes[0].catchAll {

					catchAll = t.nodes[0].nodes[0]
				}
			}

			t = t.nodes[i]
			continue WALK
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

		if t.name == "*" {
			rec.ParamsReceiver("*", path, path)
			path = ""
			break
		}

		// path 分段
		j = len(path)
		for i = 0; i < j; i++ {
			if path[i] == '/' {
				break
			}
		}

		if val, ok = t.Filter(path[:i], rw, req); !ok {
			break
		}
		if t.name != "" {
			rec.ParamsReceiver(t.name, path[:i], val)
		}
		path = path[i:]
	}

	if len(path) == 0 {

		if t.id != 0 {
			return t
		}

		// 被分割的尾斜线, 会在子节点中
		for i, c := range t.indices {
			if c == '/' && t.nodes[i].ots && t.nodes[i].id != 0 {
				return t.nodes[i]
			}
		}

		if len(t.indices) != 0 && t.indices[0] == 0 {
			// catch-all
			if t.nodes[0].path == "" {
				t = t.nodes[0].nodes[0]
			} else {
				t = t.nodes[0]
			}

			if t.id != 0 && t.name == "*" {
				rec.ParamsReceiver("*", "", "")
				return t
			}
		}
	}

	if catchAll == nil || catchAll.id == 0 {
		return nil
	}
	rec.ParamsReceiver("*", all, all)
	return catchAll
}

/**
Add 添加路由 pattern 返回相应的节点.

参数:
	path      路由 pattern. 必须以 "/" 开头.
	newFilter Filter 生成器, 如果为 nil, 用函数 NewFilter 代替.

返回:
	返回对应 path 的节点, 如果 path 重复, 返回原有节点.

注意: 因为 Add 允许重复, 调用者应该先判断 GetId() 是否为 0, 再确定是否要 SetId.
*/
func (t *Trie) Add(path string, newFilter FilterBuilder) *Trie {
	var i, j int
	var child *Trie

	if len(path) == 0 || path[0] != '/' {
		panic("rivet: Add supported only from root Trie.")
	}

	// optional trailing slashes

	ots := path[len(path)-1] == '?'

	if ots {
		path = path[:len(path)-1]
		if path == "/" {
			ots = false
		} else if len(path) < 1 || path[len(path)-1] != '/' {
			panic("rivet: invalid optional trailing slashes: " + path + "?")
		}
	}

	if newFilter == nil {
		newFilter = NewFilter
	}

	slashMax, names := analyzePath(path)

	catchAll := names["*"]
	catchAllOld := t.catchAll
	t.catchAll = t.catchAll || catchAll
	for {
		j = len(path)

		if j == 0 {
			if len(t.path) == 0 {
				panic("rivet: internal error, add a pattern group?")
			}
			break
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

			t.catchAll = t.catchAll || catchAll
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
				t.catchAll = t.catchAll || catchAll
				path = path[i:]
				continue
			}

			// 新增

			child = new(Trie)
			child.path = path[:i]
			child.perk = newPerk(child.path, newFilter)
			child.catchAll = catchAll
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
			child.id = t.id
			child.perk = t.perk
			child.path = t.path[i:]
			child.nodes = t.nodes
			child.indices = t.indices
			child.slash = slashCount(child.path)

			child.catchAll = catchAllOld
			child.names = t.names
			child.ots = t.ots
			t.ots = false
			t.names = nil

			t.id = 0
			t.perk = nil
			t.path = t.path[:i]
			t.nodes = []*Trie{child}
			t.indices = []byte{child.path[0]}

			t.slash = slashCount(t.path)
			child.slashMax = t.slashMax - t.slash
		}

		if len(path) == 0 {
			break
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
				t.catchAll = t.catchAll || catchAll
				continue
			}

			// 增加子节点
			child = new(Trie)
			child.path = path[:i]
			child.indices = []byte{} // 不能省略, 判断依据
			child.slashMax = slashCount(path)
			child.slash = slashCount(child.path)
			child.catchAll = catchAll

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

			catchAllOld = t.catchAll
			t = child

			continue
		}

		// 新子节点是模式节点
		t.catchAll = t.catchAll || catchAll

		// t 的子节点已有模式节点或分组
		if len(t.indices) != 0 && t.indices[0] == 0 {
			t = t.nodes[0]
			// 分组节点, 继续循环
			if len(t.path) == 0 {
				continue
			}
			// 分割为分组节点
			child = new(Trie)
			child.id = t.id
			child.perk = t.perk
			child.path = t.path
			child.indices = t.indices
			child.nodes = t.nodes
			child.slash = t.slash
			child.slashMax = t.slashMax
			child.catchAll = t.catchAll

			child.names = t.names
			t.names = nil

			t.id = 0
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

		// 无需在这里计算 slash, slashMax
		child = new(Trie)
		child.path = path[:i]
		child.perk = newPerk(child.path, newFilter)
		child.catchAll = t.catchAll
		path = path[i:]

		// 模式子节点索引为 0x0, 只能有一个, 位于 nodes[0]
		t.indices = append([]byte{0}, t.indices...)
		t.nodes = append([]*Trie{child}, t.nodes...)

		t = child
	}

	t.ots = ots
	t.catchAll = catchAll
	if t.names == nil {
		t.names = names
	}

	return t
}

/**
Print 输出 Trie 结构信息.

参数:
	prefix 行前缀

返回:
	节点下所有路由的数量.

输出格式:

	id max[RPG*?] 缩进'path'.slash [indices].len names

其中:
	max     此分支内最大斜线个数
	R       表示路由, GetId() 非 0.
	P       表示模式节点
	G       表示模式分组
	*       表示节点或下属节点含有 "/**" 模式.
	?       尾斜线匹配. "/?"
	.slash  节点 path 分段中的斜线数量
	indices 子节点首字符组成的索引.
	.len    子节点数量
	names   是参数名 map.


*/
func (t *Trie) Print(prefix string) (count int) {

	info := []byte{' ', ' ', ' ', ' ', ' '}

	if t.id != 0 {
		info[0] = 'R'
		count++
	}
	if t.perk != nil {
		info[1] = 'P'
	}
	if len(t.path) == 0 {
		info[2] = 'G'
	}
	if t.catchAll {
		info[3] = '*'
	}
	if t.ots {
		info[4] = '?'
	}

	fmt.Printf("%4d %3d[%v] %s'%s'.%d [%s]%d %v\n", t.id,
		t.slashMax, string(info),
		prefix, t.path, t.slash,
		string(t.indices), len(t.nodes),
		t.names,
	)

	for l := len(t.path); l >= 0; l-- {
		prefix += " "
	}
	for _, child := range t.nodes {
		count += child.Print(prefix)
	}
	if 2 == len(prefix) {
		fmt.Printf("\nRoutes: %d\n  id max[RPG*?] 'path' [indices].len names\n\n", count)
	}
	return count
}
