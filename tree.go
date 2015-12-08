package rivet

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Trie 是路由 patterns 的前缀树.
// 请务必使用 NewTrie 设定分隔符生成对象. 否则分隔符将是 0x00
type Trie struct {
	Word    interface{} // 用来保存应用数据
	pattern string      // 定值前缀或者匹配模式
	parent  *Trie       // 父节点
	childs  []*Trie     // 前缀子节点和模式匹配子节点
	matcher Matcher     // 匹配器, 模式匹配节点此值可能不为 nil
	sep     byte        // 分割符

	offset, kind, nop uint8
	// offset childs 中第一个 matcher 的偏移量(下标)
	// kind   fc "*", fd "**", fe "?", ff 定值, 0 分组, 其它表示 ":name" 长度
	// nop    至该节点参数数量 (number of parameters)

}

// NewTrie 返回以 sep 作为分割符的 Trie 根节点.
// 参数 sep 不能是 ':', '*', '?', ' '.
func NewTrie(sep byte) *Trie {
	if strings.IndexByte(":*? ", sep) >= 0 {
		panic("rivet: invalid separator")
	}
	return &Trie{sep: sep}
}

// newTrie 返回一个 *Trie.
func newTrie(sep byte) *Trie {
	return &Trie{sep: sep}
}

// IsRoot 返回 t 是否为根节点, 根节点没有父节点.
func (t *Trie) IsRoot() bool {
	return t.parent == nil
}

// IsFixed 返回 t 是否为定值节点. "*", "**" 结尾的节点属于非定值节点.
func (t *Trie) IsFixed() bool {
	return t.kind == 0xff
}

// IsCatchAll 返回 t 是否是以 "**" 结尾的 Catch-All 节点.
// Catch-All 节点不能再添加子节点.
func (t *Trie) IsCatchAll() bool {
	return t.kind == 0xfd
}

// Add 调用 AddChild 方法, build 参数为内建 Matcher 生成器.
func (t *Trie) Add(path string) *Trie {
	return t.AddChild(path, builder)
}

// AddChild 添加 path 到 t 的子节点, 返回 path 终端节点.
// 参数 build 用于生成 path 中的匹配器.
//
// TIP:
//
//   如果 t 未添加过 path, 使用 AddChild 后 t 的 pattern 仍为 "",
//   这意味着 t 是个根节点, 并允许它的子节点没有相同前缀, 比如 host 路由.
func (t *Trie) AddChild(path string, build func(string) Matcher) *Trie {
	if build == nil {
		build = builder
	}

	return t.mix(path, build, false)
}

// Mix 调用 Merge 方法, build 参数为内建 Matcher 生成器.
func (t *Trie) Mix(path string) *Trie {
	return t.Merge(path, builder)
}

// Merge 合并 path 到 t, Trie 树可能会被重构, 返回 path 终端节点.
// 如果 t 未添加过 path, Merge 后 t 会被赋予一个路由 pattern.
// 如果参数 path 非法, 可能会产生 panic, 参数 build 用于生成 path 中的匹配器.
//
// TIP:
//
//   应该对返回的 Trie.Word 进行赋值, 否则在匹配时, Trie.Word == nil 会被认为匹配失败.
func (t *Trie) Merge(path string, build func(string) Matcher) *Trie {
	if build == nil {
		build = builder
	}

	return t.mix(path, build, true)
}

func (t *Trie) mix(path string, build func(string) Matcher, merge bool) *Trie {

	if path == "" {
		return t
	}

	if t.kind == 0xfd {
		panic("rivet: Catch-All ending with the trie.")
	}

	// 分割定值前缀
	i := strings.IndexAny(path, ":*?")

	if i == -1 {
		i = len(path)
	} else if path[i] == '?' {
		// if i == 0 {
		// 	panic("rivet: invalid path: " + path)
		// }
		// i--
		if i != 0 {
			i--
		}
	}

	if i != 0 {
		if merge {
			t = t.merge(path[:i])
		} else {
			t = t.addChild(path[:i])
		}
		if i == len(path) {
			return t
		}

		path = path[i:]
		merge = false
	}

	// Matcher 节点

	if path[0] == ':' {
		i = strings.IndexByte(path, t.sep)
		if i == -1 {
			i = len(path)
		}

		return t.mixMatcher(path[:i], build, merge).mix(path[i:], build, false)
	}

	if path[0] == '*' {
		if len(path) > 1 {

			// "**", "**suffixpath"
			if path[1] == '*' {

				if strings.IndexByte(path[2:], '*') != -1 {
					panic("rivet: invalid path: " + path)
				}

				return t.mixMatcher(path, nil, merge)
			}

			// "*" || "/a/b*", "/a/b**", "/a/b*/..."
			if len(path) > 1 && path[1] != t.sep {
				panic("rivet: invalid path: " + path)
			}
		}

		return t.mixMatcher("*", nil, merge).mix(path[1:], build, false)

	}

	// "x?" || "?" 有可能问号打头
	if path[0] == '?' {
		return t.mixMatcher(path[:1], nil, merge).mix(path[1:], build, false)
	}
	return t.mixMatcher(path[:2], nil, merge).mix(path[2:], build, false)
}

// mixMatcher 增加匹配子节点, 特别的 "*", "**" 匹配总是位于最后两个
func (t *Trie) mixMatcher(pattern string, build func(string) Matcher, merge bool) *Trie {
	if merge {
		if t.pattern == pattern {
			return t
		}

		if t.pattern != "" {
			panic(fmt.Sprintf("rivet: can not Merge %v to %v", pattern, t.pattern))
		}
	} else {

		// 添加子节点
		i := int(t.offset)
		k := len(t.childs)

		for ; i < k; i++ {
			if t.childs[i].pattern == pattern ||
				(t.childs[i].pattern == "*" && pattern != "**") ||
				t.childs[i].pattern == "**" {
				break
			}
		}

		if i < k && t.childs[i].pattern == pattern {
			return t.childs[i]
		}

		// 新建
		t.childs = append(t.childs, nil)

		for k > i {
			t.childs[k] = t.childs[k-1]
			k--
		}

		n := newTrie(t.sep)
		n.parent = t
		t.childs[i] = n
		t = n
	}

	t.pattern = pattern

	if pattern[0] == ':' {
		// ":name pattern"
		k := strings.IndexByte(pattern, ' ')
		if k == -1 {
			// 只有数字的 name 当做定值处理, 比如 ":80".
			for k = len(pattern) - 1; k > 0; k-- {
				if pattern[k] < '0' || pattern[k] > 9 {
					break
				}
			}
			// 只有一个 ':' 也当做定值处理
			if k == 0 {
				if merge {
					return t.merge(pattern)
				} else {
					return t.addChild(pattern)
				}
			}

			// 优化 ":name", 免生成 matcher
			k = len(pattern)
		} else {
			t.matcher = build(pattern[k+1:])
		}

		if k > 0xfb {
			panic("rivet: parameter name too much: " + pattern)
		}

		t.kind = uint8(k)

	} else if pattern == "*" {
		t.kind = 0xfc
	} else if strings.HasPrefix(pattern, "**") {
		t.kind = 0xfd
	} else { //if len(pattern) == 2 && pattern[1] == '?' {
		// 应该可以支持个 "?"
		t.kind = 0xfe
	}

	// 计算截止到该节点的参数数量, 限制最大只能有 255 个 参数

	if t.parent != nil {
		t.nop = t.parent.nop
	}

	if t.nop == 0xff {
		panic("rivet: to much parameters")
	}

	// 只有 ":name", "**" 才增加参数
	if t.kind == 0xfd || t.kind < 0xfc {
		t.nop++
	}

	return t
}

// merge 合并定值 path, 有可能重构后代定值节点.
func (t *Trie) merge(path string) *Trie {

	if t.pattern == "" {
		t.pattern = path
		t.kind = 0xff
		return t
	}

	i := 0
	for ; i < len(t.pattern) && i < len(path); i++ {
		if t.pattern[i] != path[i] {
			break
		}
	}

	// 分割 t
	if i != 0 && i < len(t.pattern) {

		n := newTrie(t.sep)
		n.parent = t
		n.pattern, t.pattern = t.pattern[i:], t.pattern[:i]

		n.Word, t.Word = t.Word, nil
		n.childs, t.childs = t.childs, []*Trie{n}

		n.offset, n.kind, t.offset = t.offset, t.kind, 1
		n.nop = t.nop

	}

	path = path[i:]
	if path == "" {
		return t
	}

	return t.addChild(path)
}

// addChild 添加定值节点
func (t *Trie) addChild(path string) *Trie {

	c := path[0]

	offset := int(t.offset)
	i := sort.Search(offset, func(i int) bool {
		return t.childs[i].pattern[0] >= c
	})

	if i < offset && t.childs[i].pattern[0] == c {
		return t.childs[i].merge(path)
	}

	l := len(t.childs)
	t.childs = append(t.childs, nil)
	for ; i < l; l-- {
		t.childs[l] = t.childs[l-1]
	}
	n := newTrie(t.sep)
	n.kind = 0xff
	n.parent = t
	n.pattern = path
	n.nop = t.nop
	t.childs[i] = n
	t.offset++

	return n
}

// Print 输出 Trie 结构信息到 os.Stdout.
func (t *Trie) Print() {
	t.Fprint(os.Stdout)
}

// Fprint 输出 Trie 结构信息到 w.
func (t *Trie) Fprint(w io.Writer) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "word kind offset nop pattern\n")
	t.output(w, 0)
}

func (t *Trie) output(w io.Writer, count int) {

	word := " "
	if t.Word == nil {
		word = "N"
	}

	fmt.Fprintf(w, "%s %2x %3d %3d  %s%s\n", word, t.kind, t.offset, t.nop, strings.Repeat(" ", count), t.pattern)

	count += len(t.pattern)

	for _, n := range t.childs {
		n.output(w, count)
	}
}

// String 返回 Trie 的完整 pattern
func (t *Trie) String() string {
	if t.parent == nil {
		return t.pattern
	}
	return t.parent.String() + t.pattern
}

var nilBucket bucket

type bucket struct {
	req    *http.Request
	trie   *Trie
	params Params
	err    error
	pool   bool
}

// Node 调用 Match 返回 path 匹配到的节点, 忽略 http.Request, Params 和 error.
func (t *Trie) Node(path string) *Trie {
	n, _, err := t.Match(path, nil)

	if err == nil {
		return n
	}
	return nil
}

// Match 返回与 path 匹配的节点和提取到的参数.
// 参数 req 供匹配器使用.
// 返回值:
//
//   node   只有 err == nil && node != nil 才表示匹配成功.
//   params 提取到的参数.
//   err    pattern 对应的 Matcher 有可能返回错误.
//
// Catch-All 匹配到的字符串总是以 "**" 为名保存至返回的 Params 中.
func (t *Trie) Match(path string, req *http.Request) (*Trie, Params, error) {
	if path == "" {
		return nil, nil, nil
	}
	buck := &bucket{req: req}
	t.match(path, buck)
	return buck.trie, buck.params, buck.err
}

// match 负责 Matcher 匹配
func (t *Trie) match(path string, buck *bucket) {
	var (
		i      int
		val    interface{}
		childs []*Trie
	)

	switch t.kind {
	case 0xff:
		i = len(t.pattern)
		if i > len(path) || path[:i] != t.pattern {
			return
		}
	case 0xfc:
		i = strings.IndexByte(path, t.sep)
		// "/path*" 可以匹配 "/path*/", "/path/"
		if i == -1 {
			if t.Word != nil {
				buck.trie = t
			}
			return
		}
	case 0xfd: // "**", "**path/to", 仅支持后缀匹配
		if t.Word != nil {
			if len(t.pattern) != 2 {
				if !strings.HasSuffix(path, t.pattern[2:]) {
					return
				}
			}

			nop := int(t.nop)
			buck.params = make(Params, nop)
			nop--
			buck.params[nop].Name = "**"
			buck.params[nop].Source = path
			// buck.params[nop].Value = nil

			buck.trie = t

		}
		return
	case 0xfe: // ?
		if path[0] == t.pattern[0] || t.pattern[0] == '?' {
			i = 1
		}
	case 0: // 无共同前缀根节点
	default: // ":"
		i = strings.IndexByte(path, t.sep)
		if i == -1 {
			i = len(path)
		}

		if t.matcher != nil {
			if val = t.matcher.Match(path[:i], buck.req); val == nil {
				return
			}

			if val == isOk {
				val = nil
			} else if err, ok := val.(error); ok {
				buck.trie = t
				buck.err = err
				return
			}

		}
	}

	offset := int(t.offset)

	if i == len(path) {
		// path 用尽, 必定是末端

		// 处理最后一段可选尾字符匹配
		if t.Word == nil {
			childs = t.childs
			l := len(childs)
			for j := offset; j < l; j++ {
				if childs[j].kind == 0xfe && childs[j].Word != nil {
					buck.trie = childs[j]
					break
				}
			}

		} else {
			buck.trie = t
		}

	} else if buck.trie == nil {
		// t 匹配成功, 但不是终端, 递归匹配 childs
		var j, k, h int
		var c byte = path[i]
		childs = t.childs

		j = offset

		for k < j {
			h = k + (j-k)/2
			if childs[h].pattern[0] < c {
				k = h + 1
			} else {
				j = h
			}
		}

		if k < offset && childs[k].pattern[0] == c {
			childs[k].match(path[i:], buck)
		}

		if buck.trie == nil {
			k = len(childs)
			for ; offset < k; offset++ {
				childs[offset].match(path[i:], buck)

				if buck.trie != nil {
					break
				}
			}
		}
	}

	// 匹配成功增加参数
	if buck.trie != nil && t.kind < 0xfc && t.kind > 1 { // kind == 1 表示参数无命名, 只匹配不保存
		nop := int(t.nop)
		if buck.params == nil {
			buck.params = make(Params, nop)
		}

		nop--
		buck.params[nop].Name = t.pattern[1:int(t.kind)]
		buck.params[nop].Source = path[:i]
		buck.params[nop].Value = val
	}

	return
}
