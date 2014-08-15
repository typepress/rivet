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

// ParamsReceiver 逐个接受路由匹配提取到的参数.
func (p Params) ParamsReceiver(key, text string, val interface{}) {
	p[key] = val
}

/**
// ParamsNames 接收合法的参数名
func (p Params) ParamsNames(names map[string]bool) {
	if len(p) != len(names) {
		clear := len(names) == 0
		for k, _ := range p {

			if clear || !names[k] {
				delete(p, k)
			}
		}
	}
}
*/

/**
PathParams 存储原始的 URL.Path 参数.
与 Scene/NewScene 配套使用.
PathParams 符合 ParamsReceiver 接口.
*/
type PathParams map[string]string

// Get 返回 key 对应值
func (p PathParams) Get(key string) string {
	return p[key]
}

/**
// ParamsNames 接收合法的参数名
func (p PathParams) ParamsNames(names map[string]bool) {
	if len(p) != len(names) {
		clear := len(names) == 0
		for k, _ := range p {

			if clear || !names[k] {
				delete(p, k)
			}
		}
	}
}
*/

// ParamsReceiver 逐个接受路由匹配提取到的参数.
func (p PathParams) ParamsReceiver(key, text string, val interface{}) {
	p[key] = text
}

/**
ParamsReceiver 接收 URL.Path 参数.
路由匹配过程中, 当提取到参数时, 会调用参数接收函数.
事实上实例函数作为参数传递给 Trie.Match, 由 Trie.Match 调用.
*/
type ParamsReceiver interface {
	/**
	ParamsReceiver 接收参数.
	过滤前, 要先接收参数. 路由匹配时, 每此提取到一个参数就被调用一次.
	参数:
		name  参数名, "*" 代表 catch-All 模式的名字
		text URL.Path 中的原值.
		val  经 Filter 处理后的值.
	*/
	ParamsReceiver(name, text string, val interface{})

	/**
	此功能调试 bug 用
	ParamsNames 接收路由原定义中的参数名.
	路由匹配时可能发生回溯, ParamsReceiver 可能接收到多余的参数.
	匹配最后把有效的参数名回传给 ParamsNames 用于过滤参数.
	参数:
		names  以 map[string]bool 形式, name 作为 key 存储.
		此参数有可能为 nil. 表示原 pattern 中无参数可提取.

	匹配成功, 此方法总是被调用. ParamsNames 不应该更改 names.
	*/
	//ParamsNames(names map[string]bool)
}

/**
ParamsFunc 包装函数符合 ParamsReceiver 接口.
*/
type ParamsFunc func(key, text string, val interface{})

func (rec ParamsFunc) ParamsReceiver(key, text string, val interface{}) {
	rec(key, text, val)
}

/**
Filter 过滤转换 URL.Path 参数.
*/
type Filter interface {
	/**
	Filter 检验 URL.Path 中的某一段参数.
	参数 text:
		路由实例: "/blog/cat:id num 6"
			"id" 为参数名, "num" 为类型名, "6" 是参数.
		URL 实例: "/blog/cat3282"
			传递给 Filter 的参数是字符串 "3282".
			Filter 无需知道参数名, 另外处理.

	参数 rw, req:
		Filter 可能需要 req 的信息, 甚至直接写 rw.

	返回值:
		interface{} 通过检查/转换后的数据
		bool 值表示参数是否通过过滤器.
	*/
	Filter(text string,
		rw http.ResponseWriter, req *http.Request) (interface{}, bool)
}

/**
FilterFunc 包装函数符合 Filter 接口.
*/
type FilterFunc func(text string) (interface{}, bool)

func (filter FilterFunc) Filter(text string,
	_ http.ResponseWriter, _ *http.Request) (interface{}, bool) {

	return filter(text)
}

/**
FilterBuilder 是 Filter 生成器.
参数:
	className 为 Filter 类型名.
	args      为参数.
*/
type FilterBuilder func(className string, args ...string) Filter

/**
Riveter 是 Context 生成器.
*/
type Riveter func(http.ResponseWriter, *http.Request) Context

/**
Context 关联变量到 Request 上下文, 并调用 Handler.
事实上 Context 采用的是 All-In-One 的设计方式.
实现有可能未完成所有接口.
*/
type Context interface {
	// Context 要实现参数接收器
	ParamsReceiver
	// Request 返回生成 Context 的 *http.Request
	Request() *http.Request

	// Response 返回生成 Context 的 http.ResponseWriter
	Response() http.ResponseWriter

	// WriteString 方便向 http.ResponseWriter 写入 string.
	WriteString(data string) (int, error)

	//	GetParams 返回路由匹配时从 URL.Path 中提取的参数
	GetParams() Params

	/**
	PathParams 返回路由匹配时从 URL.Path 中提取的参数
	PathParams 需要与 Scene/NewScene 配套使用.
	*/
	GetPathParams() PathParams

	// Handlers 设置 Handler, 通常这只能使用一次
	Handlers(handler ...interface{})

	/**
	Invoke 处理 handler, 如果无法调用, 关联到 context.
	如果 handler 可被调用, 但是无法获取其参数, 返回 false.
	否则返回 true.
	*/
	Invoke(handler interface{}) bool

	// Next 遍历 Handlers 保存的 handler, 通过 Invoke 调用.
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
	Handlers(handler ...interface{})

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

/**
NodeBuilder 是 Node 生成器.
参数:
	id  识别号码
*/
type NodeBuilder func(id int) Node
