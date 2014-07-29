package rivet

import (
	"sort"
)

// base route
type base struct {
	rivet    Riveter
	handlers []Handler
}

func (b *base) Rivet(rivet Riveter) {
	b.rivet = rivet
}

func (b *base) Handlers(handlers ...Handler) {
	b.handlers = handlers
}

func (b *base) Apply(params Params, context Context) {
	if b == nil {
		_, req := context.Source()
		panic(req.URL.Path)
	}
	if context != nil {
		if b.rivet == nil {
			context.Invoke(params, b.handlers...)
		} else {
			b.rivet.Context(context.Source()).Invoke(params, b.handlers...)
		}
	}
}

type fixRoute struct {
	base
	url string
}

type fixRoutes map[string][]*fixRoute

func (f fixRoutes) Match(method, url string, context Context) bool {

	rs := f[method]
	n := sort.Search(len(rs), func(i int) bool {
		if len(rs[i].url) > len(url) {
			return true
		}
		if len(rs[i].url) < len(url) {
			return false
		}
		return rs[i].url >= url
	})

	if n == len(rs) || rs[n].url != url {
		if method == "*" {
			return false
		}

		if method == "HEAD" {
			rs = f["GET"]
		} else {
			rs = f["*"]
		}

		n = sort.Search(len(rs), func(i int) bool {
			if len(rs[i].url) > len(url) {
				return true
			}
			if len(rs[i].url) < len(url) {
				return false
			}
			return rs[i].url >= url
		})

		if n == len(rs) || rs[n].url != url {
			return false
		}
	}

	rs[n].Apply(nil, context)
	return true
}

func (f fixRoutes) Add(method, url string) Route {
	r := &fixRoute{url: url}

	rs := f[method]
	if len(rs) == 0 {
		f[method] = []*fixRoute{r}
		return r
	}
	n := sort.Search(len(rs), func(i int) bool {
		if len(rs[i].url) > len(url) {
			return true
		}
		if len(rs[i].url) < len(url) {
			return false
		}
		return rs[i].url >= url
	})

	if n == len(rs) || rs[n].url != url {
		rs = append(rs, nil)
		for i := len(rs) - 1; i > n; i-- {
			rs[i] = rs[i-1]
		}
		rs[n] = r
		f[method] = rs
	}

	return rs[n]
}
