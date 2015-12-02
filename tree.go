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
	Word     interface{} // 用来保存应用数据
	name     string      // 定值字符串或者参数名, 内置 "\n", "\r" 表示 "**"" 和 "*"
	pattern  string      // pattern
	parent   *Trie       // 父节点
	childs   []*Trie     // 前缀子节点
	matcher  Matcher     // 匹配器
	matchers []*Trie     // 匹配器子节点

	// 如果 matcher 为 nil 且 pattern[0] != "*" name 表示定值字符串, 否则为参数名
}

// newTrie 返回一个 *Trie.
func newTrie() *Trie {
	return new(Trie)
}

// Trie 构建通过判断 path 的前部并不断吃掉字符的过程

// Mix 传递内建的 Matcher 生成方法调用 Merge.
func (t *Trie) Mix(path string) *Trie {
	return t.Merge(path, builder)
}

// Merge 合并 path 到 t, Trie 树会被重构, 返回最终节点.
// 参数 build 用于生成 path 中的匹配.
// 如果 path 非法, 可能会产生 panic
//
// TIP:
//   应该对返回的 Trie.Word 进行赋值, 否则在匹配时, Trie.Word == nil 会被认为匹配失败.
func (t *Trie) Merge(path string, build func(string) Matcher) *Trie {

	if t.parent == nil && (path == "" || path[0] != '/') {
		panic("rivet: path must start with a forward slash.")
	}

	// 来自上级的调用可能产生空字符串
	if path == "" {
		return t
	}

	// "**"
	if t.pattern == "**" {
		panic("rivet: Catch-All ending with the trie.")
	}

	if build == nil {
		build = builder
	}

	// ":" 之前只可能是非 Matcher pattern, 定值的和段尾部包含 "*"
	i := strings.IndexAny(path, ":*")
	if i == -1 {
		i = len(path)
	}
	t = t.merge(path[:i])
	path = path[i:]
	if path == "" {
		return t
	}

	// "*", "**"
	if path[0] == '*' {
		// "*" || "/a/b*", "/a/b**", "/a/b*/..."
		if len(path) == 2 && path[1] != '*' && path[1] != '/' || len(path) > 2 && path[1] != '/' {
			panic("rivet: invalid path: " + path)
		}
		if path == "**" {
			return t.add("", "**", nil)
		}
		return t.add("", "*", nil).Merge(path[1:], build)
	}

	// Matcher pattern 只能在段尾部
	i = strings.IndexByte(path, '/')
	if i == -1 {
		i = len(path)
	}

	args := strings.SplitN(path[:i], " ", 2)
	if len(args) == 1 {
		args = append(args, "")
	}

	return t.add(args[0][1:], args[1], build).Merge(path[i:], build)
}

// add 增加匹配子节点, 特别的 "*", "**" 匹配总是位于最后两个
func (t *Trie) add(name, pattern string, build func(string) Matcher) *Trie {
	k := len(t.matchers)
	i := 0

	for ; i < k; i++ {
		if t.matchers[i].pattern == pattern ||
			(t.matchers[i].pattern == "*" && pattern != "**") ||
			t.matchers[i].pattern == "**" {
			break
		}
	}

	if i == k || t.matchers[i].pattern != pattern {
		// 新建
		t.matchers = append(t.matchers, nil)

		for k > i {
			t.matchers[k] = t.matchers[k-1]
			k--
		}

		n := newTrie()
		n.parent = t
		n.name = name
		n.pattern = pattern
		if build != nil {
			n.matcher = build(pattern)
		}
		t.matchers[i] = n
	}
	return t.matchers[i]
}

// merge 参数 path 是字面值, 可能会重构 Trie 树中所有的定值部分.
func (t *Trie) merge(path string) *Trie {

	if t.matcher == nil && t.pattern != "*" {

		if t.name == "" {
			t.name = path
			return t
		}
		// 必定有相同的部分, 至少也是个 "/"
		i := 0
		for ; i < len(t.name) && i < len(path); i++ {
			if t.name[i] != path[i] {
				break
			}
		}
		// 可能是 "*" 匹配
		if i != 0 {

			path = path[i:]

			// 分割 t
			if i < len(t.name) {

				n := newTrie()
				n.parent = t
				n.name, t.name = t.name[i:], t.name[:i]

				n.Word, t.Word = t.Word, nil
				n.matchers, t.matchers = t.matchers, []*Trie{}
				n.childs, t.childs = t.childs, []*Trie{n}
			}

			if path == "" {
				return t
			}
		}
	}

	// 简单添加定值节点
	c := path[0]
	childs := t.childs
	k := len(childs)
	i := sort.Search(k, func(i int) bool {
		return childs[i].name[0] >= c
	})

	if i == k || childs[i].name[0] != c {
		t.childs = append(t.childs, nil)
		for ; i < k; k-- {
			t.childs[k] = t.childs[k-1]
		}
		n := newTrie()
		n.name = path
		n.parent = t
		t.childs[i] = n
	}

	return t.childs[i].merge(path)
}

// Print 输出 Trie 结构信息到 os.Stdout.
func (t *Trie) Print() {
	t.output(os.Stdout, 0)
}

// Fprint 输出 Trie 结构信息到 w.
func (t *Trie) Fprint(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	t.output(w, 0)
}

func (t *Trie) output(w io.Writer, count int) {
	fmt.Fprint(w, strings.Repeat(" ", count))

	if t.name != "" {
		if t.matcher != nil {
			count++
			fmt.Fprint(w, ":")
		}

		count += len(t.name)
		fmt.Fprint(w, t.name)
	}

	if t.pattern != "" {
		if t.name != "" {
			count++
			fmt.Fprint(w, " ")
		}
		count += len(t.pattern)
		fmt.Fprint(w, t.pattern)
	}

	if t.Word == nil {
		fmt.Fprintln(w, " <nil>")
	} else {
		fmt.Fprintln(w)
	}

	for _, n := range t.childs {
		n.output(w, count)
	}

	for _, n := range t.matchers {
		n.output(w, count)
	}
}

// String 返回 Trie 的完整 pattern
func (t *Trie) String() string {
	var s string
	if t.name != "" {
		if t.matcher != nil {
			s = ":" + t.name
		} else {
			s = t.name
		}
	}

	if t.pattern != "" {
		if s != "" {
			s += " " + t.pattern
		} else {
			s += t.pattern
		}
	}

	if t.parent == nil {
		return s
	}
	return t.parent.String() + s
}

var nilBucket bucket

type bucket struct {
	trie   *Trie
	params Params
	err    error
	ok     bool
}

// Node 调用 Match 返回 path 匹配到的节点, 忽略 http.Request , Params 和 error.
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
	var (
		buck bucket
	)
	if t == nil || path == "" {
		return nil, nil, nil
	}
	if t.matcher == nil && t.pattern == "" {
		i := len(t.name)
		if len(path) >= i && path[:i] == t.name {
			if i != len(path) {
				buck = t.next(path[i:], req)
			} else if t.Word != nil {
				return t, nil, nil
			}
		}
	} else {
		buck = t.match(path, req)
	}
	if buck.ok {
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

	i = strings.IndexByte(path, '/')

	if t.matcher == nil {
		// 必定是 "**", "*" 匹配
		if t.pattern == "**" {
			if t.Word == nil {
				return
			}

			buck.params = Params{BuildParameter("**", path, path)}
			buck.trie = t
			buck.ok = true
			return
		}

		// "/path*" 可以匹配 "/path*/", "/path/"
		if i == -1 {
			if t.Word == nil {
				return
			}
			buck.trie = t
			buck.ok = true
			return
		}
		// 是否提取 "*" 匹配的部分???
	} else {

		if i == -1 {
			i = len(path)
		}

		val = t.matcher.Match(path[:i], req)
		if val == nil {
			return
		}

		if err, ok := val.(error); ok {
			buck.err = err
			buck.trie = t
			buck.ok = true
			return
		}

	}

	// 继续匹配剩余部分
	if i == len(path) {
		if t.Word == nil {
			return
		}
		buck.ok = true
		buck.trie = t
	} else {
		buck = t.next(path[i:], req)
	}

	if !buck.ok || t.name == "" || val == nil {
		return
	}

	buck.params = append(buck.params, BuildParameter(t.name, path[:i], val))

	return
}

// next 遍历前缀子节点和 Matcher 子节点
func (t *Trie) next(path string, req *http.Request) (buck bucket) {
	var (
		i, h int
		j    int = len(t.childs)
	)

	if j > 0 {
		c := path[0]
		childs := t.childs

		for i < j {
			h = i + (j-i)/2

			if childs[h].name[0] > c {
				j = h
			} else if childs[h].name[0] < c {
				i = h + 1
			} else {

				i = len(childs[h].name)
				if len(path) > i {
					buck = childs[h].next(path[i:], req)

				} else if path == childs[h].name && childs[h].Word != nil {

					buck.trie = childs[h]
					buck.ok = true
				}

				if buck.ok {
					return
				}

				break
			}
		}
	}

	j = len(t.matchers)
	for i = 0; i < j; i++ {
		if buck = t.matchers[i].match(path, req); buck.ok {
			return
		}
	}

	return
}
