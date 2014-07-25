package rivet

import (
	"net/http"
)

/**
Pattern 负责路由中的模式匹配.
*/
type Pattern interface {
	/**
	Match 匹配 URL 中的某一段.
	参数:
		路由实例: "/blog/cat<id num 6>", pattern 为 "<id num 6>"
		URL 实例: "/blog/cat3282"
			传递给 Match 的参数是 "3282".
	返回值:
		匹配处理后的数据
		bool 值表示是否匹配成功
	*/
	Match(string) (interface{}, bool)
}

/**
Riveter 用于生成 Context 实例, 需要用户实现.
*/
type Riveter interface {
	// Context 生成 Context 实例
	Context(res http.ResponseWriter, req *http.Request) Context
}

// Params 以 key/value 存储 URL 匹配到的参数
type Params map[string]interface{}

/**
Context 是实际的 http Request 处理对象.
*/
type Context interface {
	// Source 返回产生 Context 的参数
	Source() (http.ResponseWriter, *http.Request)
	/**
	Invoke 负责调用 http.Request Handler
	参数:
		params 含有路由匹配模式提取到的参数
			为 nil, 那一定是匹配失败.
			即便 len(params) 为 0 也表示匹配成功.
		handlers 由 Router 匹配得到.
			当设置了 NotFound Handler 时, 也会通过此方法传递.
			如果匹配失败, 且没有设置 NotFound Handler, 此值为 nil.
	*/
	Invoke(params Params, handlers ...Handler)
}

/**
Route 负责匹配 urls, 并支持使用不同的 Context 处理 http Request.
*/
type Route interface {
	/**
	Rivet 绑定 Riveter 实例.
	*/
	Rivet(rivet Riveter)
	/**
	Match 匹配 strings.Split(req.URL.Path, "/") 分割后的 urls.
	如果匹配成功, 通过 rivet.Context 生成 Context 实例并处理 Handler, 返回 true, 否则返回 false.
	如果没有绑定 Rivet 对象, 用 source.Context(nil,nil) 获取 Context 实例.
	如果绑定了 Rivet 对象, source.Context(rivet.Source()) 获得.
	如果 source 为 nil, 为测试模式, 不处理 Handler.
	*/
	Match(urls []string, context Context) bool
}

/**
Router 实例在 http.Request 时会通过匹配到的 Route 调用 Context.Invoke.
事实上为了能正常调用 Route.Match 方法, 生成 Router 的方法需要 Rivet 实例参数.
特别的, 如果不设定 NotFound Handler, 直接调用 http.NotFound, 不调用 Context.Invoke.
多级路由匹配次序:
	第一级
		Method 路由  指定 http.Request.Method
			如果是 HEAD 方法, 匹配失败, 尝试匹配 GET 路由.
		Any 路由     未指定 Method
	第二级
		字面匹配
		模式路由
*/
type Router interface {
	http.Handler
	// Router 也实现了 Route 接口
	Route
	/**
	Add 为 HTTP method request 添加路由
	参数:
		method  为 "*" 或者不能识别等效 Any 方法.
		pattern 为空等效 NotFound 方法, 重复定义将替换原来的 Route.
	*/
	Add(method string, pattern string, h ...Handler) Route
	// Any 为任意 HTTP method request 添加路由.
	Any(pattern string, h ...Handler) Route
	// Get 为 HTTP GET request 添加路由
	Get(pattern string, h ...Handler) Route
	// Put 为 HTTP PUT request 添加路由
	Put(pattern string, h ...Handler) Route
	// Post 为 HTTP POST request 添加路由
	Post(pattern string, h ...Handler) Route
	// Patch 为 HTTP PATCH request 添加路由
	Patch(pattern string, h ...Handler) Route
	// Head 为 HTTP HEAD request 添加路由
	Head(pattern string, h ...Handler) Route
	// Delete 为 HTTP DELETE request 添加路由
	Delete(pattern string, h ...Handler) Route
	// Options 为 HTTP OPTIONS request 添加路由
	Options(pattern string, h ...Handler) Route
	// NotFound 设置匹配失败路由, 此路由只有一个.
	NotFound(...Handler) Route
}
