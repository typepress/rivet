package rivet

import (
	"fmt"
	"net/http"
)

/**
Params 存储从 URL.Path 中提取的参数.
值可能经过 Filter 转换.
*/
type Params map[string]interface{}

// Get 返回 key 所对应值的字符串形式
func (p Params) Get(key string) (s string) {

	i, ok := p[key]

	if !ok {
		return
	}

	s, ok = i.(string)
	if ok {
		return
	}

	return fmt.Sprint(i)
}

// ParamsReceiver 逐个接受从 URL.Path 中提取的参数.
func (p Params) ParamsReceiver(key, _ string, val interface{}) {
	p[key] = val
}

/**
PathParams 存储从 URL.Path 中提取的原始参数.
与 Scene/NewScene 配套使用.
*/
type PathParams map[string]string

// Get 返回 key 对应值
func (p PathParams) Get(key string) string {
	return p[key]
}

// ParamsReceiver 逐个接受从 URL.Path 中提取的原始参数.
func (p PathParams) ParamsReceiver(key, text string, _ interface{}) {
	p[key] = text
}

/**
ParamsReceiver 接收从 URL.Path 中提取的参数.
*/
type ParamsReceiver interface {
	/**
	ParamsReceiver 逐个接受参数.
	参数:
		name 参数名, "*" 代表 catch-All 模式的名字
		text URL.Path 中的原始值.
		val  经 Filter 处理后的值.
	*/
	ParamsReceiver(name, text string, val interface{})
}

/**
ParamsFunc 包装函数符合 ParamsReceiver 接口.
*/
type ParamsFunc func(key, text string, val interface{})

func (rec ParamsFunc) ParamsReceiver(key, text string, val interface{}) {
	rec(key, text, val)
}

/**
Filter 检验, 转换 URL.Path 参数, 亦可过滤 http.Request.
*/
type Filter interface {
	/**
	Filter
	参数 text 举例:
		有路由规则 "/blog/cat:id num 6".
		实例 URL.Path "/blog/cat3282" 需要过滤.
		text 参数值是字符串 "3282".

	参数 rw, req:
		Filter 可能需要 req 的信息, 甚至直接写 rw.

	返回值:
		interface{} 通过检查/转换后的数据.
		bool 值表示是否通过过滤器.
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
	class 为 Filter 类型名.
	args  为参数.
*/
type FilterBuilder func(class string, args ...string) Filter

/**
Riveter 是 Context 生成器.
*/
type Riveter func(http.ResponseWriter, *http.Request) Context

/**
Context 是 Request 上下文, 主要负责关联变量并调用 Handler.
事实上 Context 采用 All-In-One 的设计方式,
具体实现不必未完成所有接口, 使用方法配套即可.
*/
type Context interface {
	// Context 要实现参数接收器.
	ParamsReceiver
	// Request 返回生成 Context 的 *http.Request.
	Request() *http.Request

	// Response 返回生成 Context 的 http.ResponseWriter.
	Response() http.ResponseWriter

	// WriteString 向 http.ResponseWriter 写入 data.
	WriteString(data string) (int, error)

	//	GetParams 返回路由匹配时从 URL.Path 中提取的参数.
	GetParams() Params

	/**
	PathParams 返回路由匹配时从 URL.Path 中提取的原始参数.
	需要与 Scene/NewScene 配套使用.
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
	此方法让 Node 可以拥有单独 Context.
	*/
	Riveter(riveter Riveter)

	/**
	Handlers 设置路由 Handler.
	*/
	Handlers(handler ...interface{})

	/**
	Apply 调用 Context 的 Handlers 和 Next 方法.
	如果设置了 Riveter, 可使用生成独立的 Context.
	*/
	Apply(context Context)

	/**
	Id 返回 Node 的识别 id, 0 表示 NotFound 节点.
	此值由 NodeBuilder 确定.
	*/
	Id() int
}

/**
NodeBuilder 是 Node 生成器.
参数:
	id  识别号码
*/
type NodeBuilder func(id int) Node
