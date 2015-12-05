package main

import (
	"io"
	"net/http"

	"github.com/typepress/rivet"
)

func empty() { println("empty") }

func helloWorld(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello World"))
}

func hello(params rivet.Params, w http.ResponseWriter) {
	io.WriteString(w, "Hello ")
	w.Write([]byte(params.Get("name")))
}

func catchAll(c rivet.Context) {
	c.WriteString("CatchAll:" + c.Get("**"))
}

func letGo(c *rivet.Context) {
	c.Map("Death is coming. Let's Go!")
}

func goGo(c *rivet.Context) {
	var s string

	i, has := c.Pick(rivet.TypePointerOf("string"))
	if has {
		s, _ = i.(string)
	}

	c.WriteString(c.Get("name") + "! " + s)
}

func main() {

	mux := rivet.New()

	mux.Get("/", helloWorld)
	mux.Get("/empty", empty)
	mux.Get("/hi/:name/path/to", hello)
	mux.Get("/hi/:name/path", hello)
	mux.Get("/:name", letGo, goGo)
	mux.Get("/hi/**", catchAll)

	// rivet.Router 符合 http.Handler 接口
	http.ListenAndServe(":3282", mux)
}
