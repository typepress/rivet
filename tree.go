package rivet

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Trie 是路由 patterns 的前缀树.
type Trie struct {
	Word    interface{} // 用来保存应用数据
	pattern string      // pattern
	parent  *Trie       // 父节点
	childs  []*Trie     // 前缀子节点
	matcher Matcher     // 匹配器

	offset, kind, nop uint8
	// offset childs 中第一个 matcher 的偏移量(下标)
	// kind: fc "*", fd "**", fe "?", ff 定值,  其它表示参数名在 pattern 中的右边界
	// nop    至该节点参数数量 (number of parameters)

}

// newTrie 返回一个 *Trie.
func newTrie() *Trie {
	return new(Trie)
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

	if t.pattern == "" && path[0] != '/' {
		t.kind = 0xff
		return t.mix(path, build, false, '.')
	}
	return t.mix(path, build, false, '/')
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

	if t.pattern == "" && t.offset != 0 {
		panic("rivet: invalid promiscuous mode: HostRouter + PathRouter")
	}

	return t.mix(path, build, true, '/')
}

const maxSlash = 251

func max251(max int, x uint8) uint8 {
	if x == maxSlash || max >= maxSlash {
		return maxSlash
	}
	if max > int(x) {
		return uint8(max)
	}
	return x
}

func (t *Trie) mix(path string, build func(string) Matcher, merge bool, sep byte) *Trie {

	if path == "" {
		return t
	}

	if t.pattern == "**" {
		panic("rivet: Catch-All ending with the trie.")
	}

	// 分割定值前缀
	i := strings.IndexAny(path, ":*?")

	if i == -1 {
		i = len(path)
	} else if path[i] == '?' {
		if i == 0 {
			panic("rivet: invalid path: " + path)
		}
		i--
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
		// ":name pattern" 只能在段尾部
		i = strings.IndexByte(path, sep)
		if i == -1 {
			// 特别支持 host 路由, 最后一段端口
			if sep == '.' {
				_, err := strconv.ParseUint(path[0:], 10, 0)
				if err == nil {
					return t.addChild(path)
				}
			}

			i = len(path)
		}

		return t.mixMatcher(path[:i], build, merge).mix(path[i:], build, false, sep)
	}

	if path[0] == '*' {
		if path == "**" {
			return t.mixMatcher("**", nil, merge)
		}
		// "*" || "/a/b*", "/a/b**", "/a/b*/..."
		if len(path) > 1 && path[1] != sep {
			panic("rivet: invalid path: " + path)
		}
		return t.mixMatcher("*", nil, merge).mix(path[1:], build, false, sep)

	}

	// "x?"
	return t.mixMatcher(path[:2], nil, merge).mix(path[2:], build, false, sep)
}

// mixMatcher 增加匹配子节点, 特别的 "*", "**" 匹配总是位于最后两个
func (t *Trie) mixMatcher(pattern string, build func(string) Matcher, merge bool) *Trie {
	if merge {
		if t.pattern == pattern {
			return t
		}

		if t.pattern == "" {
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

		n := newTrie()
		n.parent = t
		t.childs[i] = n
		t = n
	}

	t.pattern = pattern

	if pattern[0] == ':' {
		// ":name pattern"
		k := strings.IndexByte(pattern, ' ')
		if k == -1 {
			// 优化 ":name", 免生成 matcher
			k = len(pattern)
		} else {
			t.matcher = build(pattern[k+1:])
		}

		if k > 0xfc {
			panic("rivet: parameter name is too long: " + pattern)
		}

		t.kind = uint8(k)

	} else if pattern == "*" {
		t.kind = 0xfc
	} else if pattern == "**" {
		t.kind = 0xfd
	} else if len(pattern) == 2 && pattern[1] == '?' {
		t.kind = 0xfe
	} else {
		panic("rivet: invalid pattern: " + pattern)
	}

	return t.countParams()
}

// 计算截止到该节点的参数数量
func (t *Trie) countParams() *Trie {

	if t.parent != nil {
		t.nop = t.parent.nop
		if t.nop == 0xff {
			return t
		}
	}

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

		n := newTrie()
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
	n := newTrie()
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
	fmt.Fprintln(w, "word offset kind nop pattern\n")
	t.output(w, 0)
}

func (t *Trie) output(w io.Writer, count int) {

	word := " "
	if t.Word == nil {
		word = "N"
	}

	fmt.Fprintf(w, "%s %2x %2x %2x  %s%s\n", word, t.offset, t.kind, t.nop, strings.Repeat(" ", count), t.pattern)

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
	add    bool // 是否用 append 加入参数
	sep    byte
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
func (t *Trie) Match(path string, req *http.Request) (node *Trie, params Params, err error) {
	var buck *bucket
	if path == "" {
		return // nil, nil, nil
	}

	if t.pattern == "" && path[0] != '/' {
		buck = &bucket{req: req, sep: '.'}
	} else {
		buck = &bucket{req: req, sep: '/'}
	}

	t.match(path, buck)
	return buck.trie, buck.params, buck.err
}

// match 负责 Matcher 匹配
func (t *Trie) match(path string, buck *bucket) {
	var (
		i   int
		val interface{}
	)

	switch t.kind {
	case 0xff:
		i = len(t.pattern)
		if i > len(path) || path[:i] != t.pattern {
			return
		}
	case 0xfc:
		i = strings.IndexByte(path, buck.sep)
		// "/path*" 可以匹配 "/path*/", "/path/"
		if i == -1 {
			if t.Word != nil {
				buck.trie = t
			}
			return
		}
	case 0xfd:
		if t.Word != nil {
			nop := int(t.nop)
			if nop == 255 {
				buck.add = true
				buck.params = Params{Argument{"**", path, path}}
			} else {
				buck.params = make(Params, nop)
				buck.params[nop-1] = Argument{"**", path, path}
			}
			buck.trie = t
		}
		return
	case 0xfe: // ?
		if path[0] == t.pattern[0] {
			i = 1
		}
	default: // ":"
		i = strings.IndexByte(path, buck.sep)
		if i == -1 {
			i = len(path)
		}

		if t.matcher != nil {
			if val = t.matcher.Match(path[:i], buck.req); val == nil {
				return
			}

			if err, ok := val.(error); ok {
				buck.trie = t
				buck.err = err
				return
			}

		}

		if i == len(path) && t.Word != nil {
			nop := int(t.nop)
			if nop == 255 {
				buck.add = true
				buck.params = Params{Argument{t.pattern[1:int(t.kind)], path[:i], val}}
			} else {
				buck.params = make(Params, nop)
				buck.params[nop-1] = Argument{t.pattern[1:int(t.kind)], path[:i], val}
			}
			buck.trie = t
			return
		}
	}

	// pattern 已经匹配成功
	offset := int(t.offset)

	if i != len(path) {

		// 递归匹配 childs
		c := path[i]
		j := 0
		for j < offset {
			h := j + (offset-j)/2
			if t.childs[h].pattern[0] < c {
				j = h + 1
			} else {
				offset = h
			}
		}

		if j < offset && t.childs[j].pattern[0] == c {
			t.childs[j].match(path[i:], buck)
		}

		if buck.trie == nil {

			k := len(t.childs)
			for j := offset; j < k; j++ {
				t.childs[j].match(path[i:], buck)
				if buck.trie != nil {
					break
				}
			}
		}

	} else {

		// 最后一段
		if t.Word == nil {
			// 特别处理可选尾斜线
			if offset != 0 && buck.sep == '/' {
				l := len(t.childs)
				for j := offset; j < l; j++ {
					if t.childs[j].pattern == "/?" && t.childs[j].Word != nil {
						buck.trie = t.childs[j]
						break
					}
				}
			}

			if buck.trie == nil {
				return
			}
		} else {
			buck.trie = t
		}
	}

	if buck.trie != nil && t.kind < 0xfc && i != 0 { // i == 0 表示 pattern == ""

		nop := int(t.nop)

		if buck.add || nop == 255 {
			buck.add = true
			buck.params = append(buck.params, Argument{t.pattern[1:int(t.kind)], path[:i], val})
			return
		}

		if buck.params == nil {
			buck.params = make(Params, nop)
		}

		buck.params[nop-1] = Argument{t.pattern[1:int(t.kind)], path[:i], val}
	}

	return
}
