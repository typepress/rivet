package rivet

import (
	"fmt"
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

// Params
type Params map[string]interface{}

// Get 返回 key 所对应值的字符串形式
func (p Params) Get(key string) string {
	for k, i := range p {
		if k == key {
			return fmt.Sprint(i)
		}
	}
	return ""
}

/**
Riveter 用于构建 Context.
*/
type Riveter func(http.ResponseWriter, *http.Request, Params) Context

/**
Context 支持关联变量到上下文.
*/
type Context interface {
	// Request 返回产生 Context 的 *http.Request
	Request() *http.Request

	// Response 返回产生 Context 的 http.ResponseWriter
	Response() http.ResponseWriter

	// WriteString 方便向 http.ResponseWriter 写入 string.
	WriteString(data string) (int, error)

	//	PathParams 返回路由匹配时从 URL.Path 中提取的参数
	PathParams() Params

	// Handlers 负责设置 Handler, 通常这只能使用一次
	Handlers(...Handler)

	// Next 负责调用 Handler
	Next()

	// 以变量 v 的类型标识为 key , 关联 v 到 context.
	Map(v interface{})

	// 以指定的类型标识 t 为 key , 关联 v 到 context.
	MapTo(v interface{}, t uint)

	// 以类型标识 t 为 key, 获取关联到 context 的变量.
	Get(t uint) interface{}
}

// Node 保存路由 Handler, 并负责调用 Context
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
	Apply 调用 Context 的 Next() 方法.
	如果设置了 Riveter, 生成新 Context 并调用新的 Next().
	*/
	Apply(context Context)

	/**
	Id 返回 Node 的识别 id, 特别的 0 表示 NotFound 节点.
	此 id 在生成的时候确定.
	*/
	Id() int
}
