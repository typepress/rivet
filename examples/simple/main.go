package main

import (
	"net/http"

	"github.com/typepress/rivet"
)

func empty() { println("empty") }

func HelloWorld(w http.ResponseWriter) {
	w.Write([]byte("Hello World"))
}

func Hello(params rivet.Params, w http.ResponseWriter) {
	w.Write([]byte("Hello " + params["name"].(string)))
}

func CatchAll(c rivet.Context) {
	c.WriteString("CatchAll:" + c.PathParams().Get("*"))
}

func Go(c rivet.Injector) {
	c.Map("Death is coming. Let's Go!")
}

func GoGo(c rivet.Injector) {
	// 获取 string 类型标识
	id := rivet.TypeIdOf("string")

	c.WriteString(c.PathParams().Get("name") + "! " +
		c.Get(id).(string)) // 类型转换
}

func main() {

	// 传递 nil, NewRouter 使用内部的 Rivet 实现
	mux := rivet.NewRouter(nil)

	mux.Get("/", HelloWorld)
	mux.Get("/empty", empty)
	mux.Get("/hi/:name string 5/path/to", Hello)
	mux.Get("/hi/:name string 5/path", Hello)
	mux.Get("/:name", Go, GoGo)
	mux.Get("/hi/**", CatchAll)

	// 调试输出 trie 结构, 便于观察
	_, route := mux.Match("GET", "/")
	route.(*rivet.Trie).Print("")

	// rivet.Router 符合 http.Handler 接口
	http.ListenAndServe(":3000", mux)
}
