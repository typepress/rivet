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
type Trie struct {
	Word    interface{} // 用来保存应用数据
	pattern string      // pattern
	parent  *Trie       // 父节点
	childs  []*Trie     // 前缀子节点
	matcher Matcher     // 匹配器
	offset  int         // 静态节点的个数, 第一个 matcher 子节点的下标
	kind    int         // 0 定值, 1 "*", 2 "**", 3 "?", 其它值 kind>>2 表示参数名的长度

	// 如果 matcher 为 nil 且 pattern[0] != "*" 表示定值字符串, 否则为匹配
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
	return t.kind == 0
}

// IsCatchAll 返回 t 是否是以 "**" 结尾的 Catch-All 节点.
// Catch-All 节点不能再添加子节点.
func (t *Trie) IsCatchAll() bool {
	return t.kind == 2
}

// Add 传递内建 Matcher 生成器给 MergeChildren 方法.
func (t *Trie) Add(path string) *Trie {
	return t.MergeChildren(path, builder)
}

// MergeChildren 合并 path 到 t 的子节点, 参数 build 用于生成 path 中的匹配器.
func (t *Trie) MergeChildren(path string, build func(string) Matcher) *Trie {
	if build == nil {
		build = builder
	}
	return t.mix(path, build, t.pattern == "")
}

// Mix 传递内建 Matcher 生成器给 Merge 方法.
func (t *Trie) Mix(path string) *Trie {
	return t.Merge(path, builder)
}

// Merge 合并 path 到 t, Trie 树会被重构, 返回最终节点.
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

	if t.pattern == "**" {
		panic("rivet: Catch-All ending with the trie.")
	}

	// 分割定值节点
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
		return t.mix(path[i:], build, false)
	}

	// Matcher 节点

	if path[0] == ':' {
		// ":name pattern" 只能在段尾部
		i = strings.IndexByte(path, '/')
		if i == -1 {
			i = len(path)
		}

		return t.mixMatcher(path[:i], build, merge).mix(path[i:], build, false)
	}

	if path[0] == '*' {
		if path == "**" {
			return t.mixMatcher("**", nil, merge)
		}
		// "*" || "/a/b*", "/a/b**", "/a/b*/..."
		if len(path) > 1 && path[1] != '/' {
			panic("rivet: invalid path: " + path)
		}
		return t.mixMatcher("*", nil, merge).mix(path[1:], build, false)

	}

	// "x?"
	return t.mixMatcher(path[:2], nil, merge).mix(path[2:], build, false)
}

// mixMatcher 增加匹配子节点, 特别的 "*", "**" 匹配总是位于最后两个
func (t *Trie) mixMatcher(pattern string, build func(string) Matcher, merge bool) *Trie {
	if merge && t.pattern == pattern {
		return t
	}

	if !merge && t.pattern != "" {
		i := int(t.offset)
		k := len(t.childs)

		for ; i < k; i++ {
			if t.childs[i].pattern == pattern ||
				(t.childs[i].pattern == "*" && pattern != "**") ||
				t.childs[i].pattern == "**" {
				break
			}
		}

		if i == k || t.childs[i].pattern != pattern {
			// 新建
			t.childs = append(t.childs, nil)

			for k > i {
				t.childs[k] = t.childs[k-1]
				k--
			}

			n := newTrie()
			n.parent = t
			t.childs[i] = n
		}
		t = t.childs[i]
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

		if k > 63 {
			panic("rivet: parameter name is too long: " + pattern)
		}

		t.kind = k << 2

	} else if pattern == "*" {
		t.kind = 1
	} else if pattern == "**" {
		t.kind = 2
	} else if len(pattern) == 2 && pattern[1] == '?' {
		t.kind = 3
	} else {
		panic("rivet: invalid pattern: " + pattern)
	}

	return t
}

// merge 合并定值 path, 有可能重构后代定值节点.
func (t *Trie) merge(path string) *Trie {

	if t.pattern == "" {
		t.pattern = path
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
		n.offset, t.offset = t.offset, 1
		n.childs, t.childs = t.childs, []*Trie{n}

		// "abc".mix("a")
	}

	path = path[i:]
	if path == "" {
		return t
	}

	return t.addChild(path)
}

func (t *Trie) addChild(path string) *Trie {

	// 简单添加定值节点
	c := path[0]

	i := sort.Search(t.offset, func(i int) bool {
		return t.childs[i].pattern[0] >= c
	})

	if i < t.offset && t.childs[i].pattern[0] == c {
		return t.childs[i].merge(path)
	}

	l := len(t.childs)
	t.childs = append(t.childs, nil)
	for ; i < l; l-- {
		t.childs[l] = t.childs[l-1]
	}
	n := newTrie()
	n.parent = t
	n.pattern = path
	t.childs[i] = n
	t.offset++

	return n
}

// Print 输出 Trie 结构信息到 os.Stdout.
func (t *Trie) Print() {
	t.output(os.Stdout, 0)
}

// Fprint 输出 Trie 结构信息到 w.
func (t *Trie) Fprint(w io.Writer) {
	if w == nil {
		return
	}
	t.output(w, 0)
}

func (t *Trie) output(w io.Writer, count int) {
	fmt.Fprint(w, strings.Repeat(" ", count))

	count += len(t.pattern)
	s := 50 - count
	if s < 0 {
		s = 0
	}

	if t.Word == nil {
		fmt.Fprintln(w, t.pattern, strings.Repeat(" ", s), t.offset, t.kind>>2, "nil")
	} else {
		fmt.Fprintln(w, t.pattern, strings.Repeat(" ", s), t.offset, t.kind>>2)
	}

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
	trie   *Trie
	params Params
	err    error
	ok     bool
}

// Node 调用 Match 返回 path 匹配到的节点, 忽略 http.Request, Params 和 error.
func (t *Trie) Node(path string) (n *Trie) {
	n, _, _ = t.Match(path, nil)
	return
}

// Match 返回与 path 匹配的节点和提取到的参数.
// 参数 req 供匹配器使用.
// 返回值:
//
//   *Trie  通常该值非 nil 且 Trie.Word 非 nil 才表示匹配成功.
//   Params 提取到的参数.
//   error  pattern 对应的 Matcher 有可能返回错误.
//
// Catch-All 匹配到的字符串总是以 "**" 为名保存至返回的 Params 中.
func (t *Trie) Match(path string, req *http.Request) (*Trie, Params, error) {
	if path == "" {
		return nil, nil, nil
	}

	if buck := t.match(path, req); buck.ok {
		return buck.trie, buck.params, buck.err
	}
	return nil, nil, nil
}

// match 负责 Matcher 匹配
func (t *Trie) match(path string, req *http.Request) (buck bucket) {
	var (
		i   int
		val interface{}
	)

	if t.pattern[0] == ':' {
		i = strings.IndexByte(path, '/')
		if i == -1 {
			i = len(path)
		}

		if t.matcher == nil {
			val = path[:i]
		} else {
			if val = t.matcher.Match(path[:i], req); val == nil {
				return
			}

			if err, ok := val.(error); ok {
				buck.err = err
				buck.trie = t
				buck.ok = true
				return
			}
		}
	} else {
		switch t.kind {
		case 0:
			i = len(t.pattern)
			if i > len(path) || path[:i] != t.pattern {
				return
			}
		case 1:
			i = strings.IndexByte(path, '/')
			// "/path*" 可以匹配 "/path*/", "/path/"
			if i == -1 {
				if t.Word == nil {
					return
				}
				buck.trie = t
				buck.ok = true
				return
			}
		case 2:
			if t.Word == nil {
				return
			}

			buck.params = Params{Argument{"**", path, path}}
			buck.trie = t
			buck.ok = true
			return
		case 3: // ?
			if path[0] == t.pattern[0] {
				i = 1
			}
		}
	}

	if i == len(path) {
		if t.Word == nil {
			return
		}
		buck.ok = true
		buck.trie = t
	} else {
		buck = t.next(path[i:], req)
	}

	if t.kind > 4 && buck.ok && buck.err == nil && val != nil {
		buck.params = append(buck.params, Argument{t.pattern[1 : t.kind>>2], path[:i], val})
	}

	return
}

func (t *Trie) next(path string, req *http.Request) (buck bucket) {
	var k, h int

	c := path[0]
	j := t.offset
	childs := t.childs

	for k < j {
		h = k + (j-k)/2
		if childs[k].pattern[0] < c {
			j = h
		} else if childs[k].pattern[0] > c {
			k = h + 1
		} else {

			j = len(childs[k].pattern)
			if len(path) >= j && path[:j] == childs[k].pattern {

				if len(path) == j {
					if childs[k].Word != nil {
						buck.ok = true
						buck.trie = childs[k]
						return
					}
					break
				}

				buck = childs[k].next(path[j:], req)
				if buck.ok {
					return
				}
			}
			break
		}
	}

	k = len(childs)
	for ; j < k; j++ {
		buck = childs[j].match(path, req)
		if buck.ok {
			return
		}
	}
	return
}
