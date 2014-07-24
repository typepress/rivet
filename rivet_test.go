package rivet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testRivet struct{}
type testContext struct {
	res http.ResponseWriter
	req *http.Request
}

func (r testRivet) Context(res http.ResponseWriter, req *http.Request) Context {
	return &testContext{res, req}
}
func (c *testContext) Source() (http.ResponseWriter, *http.Request) {
	return c.res, c.req
}
func (c *testContext) Run(params Params, handlers ...Handler) {
	for _, h := range handlers {
		switch fn := h.(type) {
		default:
			panic(h)
		case http.HandlerFunc:
			fn(c.res, c.req)
		case func(http.ResponseWriter, *http.Request):
			fn(c.res, c.req)
		case func(Params):
			fn(params)
		case func(*http.Request):
			fn(c.req)
		}
	}
}

func Test_Routing(t *testing.T) {
	mux := NewRouter(testRivet{}).(*router)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	want := func(g interface{}, w interface{}) {
		assert(t, g, w)
	}

	var restr string
	Do := func(res, method, urlStr string) {
		restr = res
		req, _ := http.NewRequest(method, srv.URL+urlStr, nil)
		http.DefaultClient.Do(req)
	}

	result := ""
	mux.Post("/bar/:cat", func(params Params) {
		want(params["cat"], "bat")
		result += restr + ":cat"
	})

	mux.Get("/foo", func(req *http.Request) {
		result += restr + "foo"
	})
	mux.Get("/foo/*", func(req *http.Request) {
		result += restr + "*"
	})
	want(len(mux.get.routes), len(mux.get.literal))

	mux.Get("/foo/prefix*", func(req *http.Request) {
		result += restr + "prefix*"
	})
	want(len(mux.get.routes), 2)
	want(mux.get.routes[0].begin, mux.get.routes[1].begin)

	mux.Post("/foo/post:id", func(params Params) {
		want(params["id"], 6)
		result += restr + "post"
	})

	mux.Patch("/bar/:id", func(params Params) {
		want(params["id"], "foo")
		result += restr + "id"
	})

	mux.Any("/any/foo<ID uint>", func(params Params) {
		want(params["ID"], 6000)
		result += restr + "ID"
	})

	Do("1", "POST", "/bar/bat")
	Do("2", "GET", "/foo")
	Do("3", "GET", "/foo/start")
	Do("4", "GET", "/foo/prefixstart")
	Do("5", "PATCH", "/bar/foo")
	Do("6", "POST", "/foo/post6")
	Do("7", "POST", "/any/foo6000")
	Do("8", "GET", "/any/foo6000")

	want(result, "1:cat2foo3*4prefix*5id6post7ID8ID")
}

func assert(t *testing.T, got interface{}, want interface{}) {
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatal(
			"\ngot :", got,
			"\nwant:", want,
		)
	}
}
