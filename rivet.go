package rivet

import (
	"fmt"
	"net/http"
)

// Params
type Params map[string]interface{}

// Get 返回 key 所对应值的字符串形式
func (p Params) Get(key string) string {

	i, ok := p[key]

	if !ok {
		return ""
	}

	s, ok := i.(string)
	if ok {
		return s
	}

	return fmt.Sprint(i)
}

/**
Riveter 是 Context 生成器.
*/
type Riveter func(http.ResponseWriter, *http.Request, Params) Context

/**
NodeBuilder 是 Node 生成器.
参数:
	id  识别号码
	key 用于过滤 URL.Path 参数名, 缺省全通过
*/
type NodeBuilder func(id int, key ...string) Node

/**
FilterBuilder 是 Filter 生成器.
参数:
	class 为 Filter 类型名.
	args  为参数.
*/
type FilterBuilder func(class string, args ...string) Filter

/**
Filter 过滤转换 URL.Path 参数.
*/
type Filter interface {
	/**
	Filter 检验 URL.Path 中的某一段参数.
	参数:
		路由实例: "/blog/cat:id num 6"
			"id" 为参数名, "num" 为类型名, "6" 是参数.
		URL 实例: "/blog/cat3282"
			传递给 Filter 的参数是字符串 "3282".
			Filter 无需知道参数名, 另外处理.
	返回值:
		interface{} 通过检查/转换后的数据
		bool 值表示是否通过检查成功
	*/
	Filter(string) (interface{}, bool)
}

/**
Context 关联变量到 Request 上下文, 并调用 Handler.
*/
type Context interface {
	// Request 返回生成 Context 的 *http.Request
	Request() *http.Request

	// Response 返回生成 Context 的 http.ResponseWriter
	Response() http.ResponseWriter

	// WriteString 方便向 http.ResponseWriter 写入 string.
	WriteString(data string) (int, error)

	//	Params 返回路由匹配时从 URL.Path 中提取的参数
	Params() Params

	// Handlers 设置 Handler, 通常这只能使用一次
	Handlers(...Handler)

	// Next 负责调用 Handler
	Next()

	// Map 等同 MapTo(v, TypeIdOf(v))
	Map(v interface{})

	/**
	MapTo 以 t 为 key 把变量 v 关联到 context. 相同 t 值只保留一个.
	*/
	MapTo(v interface{}, t uint)

	/**
	Get 以类型标识 t 为 key, 获取关联到 context 的变量.
	如果未找到, 返回 nil.
	*/
	Get(t uint) interface{}
}

/**
Node 保存路由 Handler, 并调用 Context 的 Handlers 和 Next 方法.
*/
type Node interface {
	/**
	Riveter 设置 Riveter.
	此方法使得 Node 可以使用不同的 Context.
	*/
	Riveter(riveter Riveter)

	/**
	Handlers 设置路由 Handler.
	*/
	Handlers(handler ...Handler)

	/**
	Apply 调用 Context 的 Handlers 和 Next 方法.
	如果设置了 Riveter, 使用 Riveter 生成新 Context.
	*/
	Apply(context Context)

	/**
	Id 返回 Node 的识别 id, 0 表示 NotFound 节点.
	此 id 在生成的时候确定.
	*/
	Id() int
}
