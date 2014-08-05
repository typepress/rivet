package rivet

import (
	"net/http"
)

/**
Pattern 用于路由中的模式匹配接口.
*/
type Pattern interface {
	/**
	Match 匹配 URL 中的某一段.
	参数:
		路由实例: "/blog/cat:id num 6", pattern 为 ":id num 6"
		URL 实例: "/blog/cat3282"
			传递给 Match 的参数是字符串 "3282".
	返回值:
		匹配处理后的数据
		bool 值表示是否匹配成功
	*/
	Match(string) (interface{}, bool)
}

/**
Riveter 用于生成 Context 实例.
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
Injector 扩展 Context, 支持关联变量到 context.
*/
type Injector interface {
	Context

	// 以变量 v 的类型标识为 key , 关联 v 到 context.
	Map(v interface{})

	// 以指定的类型标识 t 为 key , 关联 v 到 context.
	MapTo(v interface{}, t uint)

	// 以类型标识 t 为 key, 获取关联到 context 的变量.
	Get(t uint) interface{}
}

/**
Route 负责通过 Context 调用 handlers, 处理 http Request.
*/
type Route interface {
	/**
	Rivet 绑定 Riveter 实例.
	此方法使得 Route 可以使用不同的 Context 实现.
	*/
	Rivet(rivet Riveter)
	/**
	Handlers 设置 Route Handler,
	如果 Router 生成 Route 的时候没有设置, 或者需要重新设置的话.
	*/
	Handlers(handlers ...Handler)
	/**
	Apply 以 params 和设置的 handlers 为参数调用 context.Invoke,
	如果绑定了 Riveter, 那么生成新的 context.
	*/
	Apply(params Params, context Context)
}

/**
Router 通过匹配到的 Route 调用 Context.Invoke.
事实上为了能正常调用 Route.Match 方法, 生成 Router 的方法需要 Rivet 实例参数.
如果不设定 NotFound Handler, 直接调用 http.NotFound, 不调用 Context.Invoke.
*/
type Router interface {
	http.Handler
	/**
	Add 为 HTTP method request 添加路由
	参数:
		method  "*" 等效 Any. 其它值不做处理, 直接和 http.Request.Method 比较.
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

	/**
	Match 以 method, urlPath 匹配路由.
	返回从 urlPath 提取的 pattern 参数和对应的路由.
	匹配失败返回  nil, nil.
	*/
	Match(method, urlPath string) (Params, Route)
}
