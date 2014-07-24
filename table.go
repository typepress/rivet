package rivet

import (
	"sort"
)

type table struct {
	routes  []*route
	literal map[string]*baseRoute //字面路由
}

func newTable() *table {
	t := &table{}
	t.routes = []*route{}
	t.literal = map[string]*baseRoute{}
	return t
}

/**
addRoute 路由表二分法插入算法
如果返回 -1 表示内部错误, 这不应该发生.

以 "/path/to/prefix<name class>suffix/foo/prefix*suffix/*" 为例
以 urls 命名 "/" 分段后得到字符串数组:

["","path","to","prefix<name class>suffix","foo","prefix*suffix","*"]

其中包含 "<",">","*" 的是模式匹配定义, 其他是字面值.
urls 是生成 Route 的源头, 对 ruls 进行组合, 排序, 匹配实现路由.

匹配次序: urls[0], urls 的数量, 字面值顺序匹配, 模式顺序匹配
*/
func (t *table) addRoute(r *route) {
	rs := t.routes
	size := len(rs)
	if size == 0 {
		t.routes = []*route{r}
		return
	}
	begin := -1
	// func 返回 true 向 left 二分, 否则向 right 二分
	n := sort.Search(size, func(i int) bool {
		if r == nil {
			return false
		}
		n := rs[i].cmp(r)
		if n == 0 {
			begin = rs[i].begin
			// 替换重复
			if rs[i].prefix == r.prefix {
				r.begin = rs[i].begin
				rs[i] = r
				r = nil
			}
			//n = rs[i].cmpattern(r)
		}
		// r < i , 左移
		return n == 1
	})

	// 被替换
	if r == nil {
		return
	}

	// 设置 index
	if begin == -1 {
		begin = n
	}
	r.begin = begin

	// 如果要加锁, 应该从这里开始

	rs = append(rs, nil)
	if n != size {
		copy(rs[n+1:], t.routes[n:])
	}
	rs[n] = r
	for i := n + 1; i <= size; i++ {
		if rs[i].begin != begin {
			rs[i].begin++
		}
	}
	t.routes = rs
}

// Match 方法只进行模式匹配, 不做字面路由匹配
func (t *table) Match(urls []string, context Context) bool {
	rs := t.routes

	// 最大值
	n := -1
	sort.Search(len(rs), func(i int) bool {
		z := rs[i].match(urls)
		if z == 0 {
			n = i
		}
		return z == 1
	})

	if n == -1 {
		return false
	}
	// 贪心比较
	begin := rs[n].begin
	for ; n >= 0; n-- {
		if rs[n].begin != begin {
			break
		}

		if rs[n].apply(urls, context) {
			return true
		}
	}
	return false
}

func (t *table) MatchUrl(url string, context Context) bool {
	r := t.literal[url]
	if r == nil {
		return false
	}
	r.Match(nil, context)
	return true
}
