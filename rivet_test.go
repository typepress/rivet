package rivet

import "testing"

var routes = []string{
	"/feeds",
	"/notifications",
	"/notifications/threads/:id",
	"/notifications/threads/:id/subscription",
	"/repos/:owner/:repo/notifications",
	"/repos/:owner/:repo/stargazers",
	"/repos/:owner/:repo/subscription",
	"/user/starred",
	"/user/starred/:owner/:repo",
	"/user/subscriptions",
	"/user/subscriptions/:owner/:repo",
	"/users/:user/events",
	"/users/:user/events/orgs/:org",
	"/users/:user/events/public",
	"/users/:user/received_events",
	"/users/:user/received_events/public",
	"/users/:user/starred",
	"/users/:user/subscriptions",
	"/hi",
	"/hi/**",
	"/hi/path/to",
	"/hi/:name/to",
	"/:name",
	"/:name/**",
	"/:name/path",
	"/:name/path*/to",
	"/:name/path/**",
	"/:name/*/to",
	`/just:id ^\d+$`,
	"/tips?",
	"/slash/?",
	"/suffix**.go",
}

var urls = []string{
	"/feeds",
	"/notifications",
	"/notifications/threads/id",
	"/notifications/threads/id/subscription",
	"/repos/owner/repo/notifications",
	"/repos/owner/repo/stargazers",
	"/repos/owner/repo/subscription",
	"/user/starred",
	"/user/starred/owner/repo",
	"/user/subscriptions",
	"/user/subscriptions/owner/repo",
	"/users/user/events",
	"/users/user/events/orgs/org",
	"/users/user/events/public",
	"/users/user/received_events",
	"/users/user/received_events/public",
	"/users/user/starred",
	"/users/user/subscriptions",
	"/hi",
	"/hi/catch/All",
	"/hi/path/to",
	"/hi/name/to",
	"/name",
	"/name/catchAll",
	"/name/path",
	"/name/path_Star/to",
	"/name/path/star/star",
	"/name/star/to",
	"/just998",
	"/tip",
	"/slash/",
	"/suffix/path/to.go",
}

var hostRoutes = []string{
	"a.b.c",
	"a.b.c:80",
	":name.b.c",
	"api.:name.b.c",
	":id uint.a.b.c",
	"id*.a.b.c",
}

var hosts = []string{
	"a.b.c",
	"a.b.c:80",
	"api.b.c",
	"api.a.b.c",
	"123.a.b.c",
	"id123.a.b.c",
}

var staticRoutes = []string{
	"/",
	"/cmd.html",
	"/code.html",
	"/contrib.html",
	"/contribute.html",
	"/debugging_with_gdb.html",
	"/docs.html",
	"/effective_go.html",
	"/files.log",
	"/gccgo_contribute.html",
	"/gccgo_install.html",
	"/go-logo-black.png",
	"/go-logo-blue.png",
	"/go-logo-white.png",
	"/go1.1.html",
	"/go1.2.html",
	"/go1.html",
	"/go1compat.html",
	"/go_faq.html",
	"/go_mem.html",
	"/go_spec.html",
	"/help.html",
	"/ie.css",
	"/install-source.html",
	"/install.html",
	"/logo-153x55.png",
	"/Makefile",
	"/root.html",
	"/share.png",
	"/sieve.gif",
	"/tos.html",
	"/articles/",
	"/articles/go_command.html",
	"/articles/index.html",
	"/articles/wiki/",
	"/articles/wiki/edit.html",
	"/articles/wiki/final-noclosure.go",
	"/articles/wiki/final-noerror.go",
	"/articles/wiki/final-parsetemplate.go",
	"/articles/wiki/final-template.go",
	"/articles/wiki/final.go",
	"/articles/wiki/get.go",
	"/articles/wiki/http-sample.go",
	"/articles/wiki/index.html",
	"/articles/wiki/Makefile",
	"/articles/wiki/notemplate.go",
	"/articles/wiki/part1-noerror.go",
	"/articles/wiki/part1.go",
	"/articles/wiki/part2.go",
	"/articles/wiki/part3-errorhandling.go",
	"/articles/wiki/part3.go",
	"/articles/wiki/test.bash",
	"/articles/wiki/test_edit.good",
	"/articles/wiki/test_Test.txt.good",
	"/articles/wiki/test_view.good",
	"/articles/wiki/view.html",
	"/codewalk/",
	"/codewalk/codewalk.css",
	"/codewalk/codewalk.js",
	"/codewalk/codewalk.xml",
	"/codewalk/functions.xml",
	"/codewalk/markov.go",
	"/codewalk/markov.xml",
	"/codewalk/pig.go",
	"/codewalk/popout.png",
	"/codewalk/run",
	"/codewalk/sharemem.xml",
	"/codewalk/urlpoll.go",
	"/devel/",
	"/devel/release.html",
	"/devel/weekly.html",
	"/gopher/",
	"/gopher/appenginegopher.jpg",
	"/gopher/appenginegophercolor.jpg",
	"/gopher/appenginelogo.gif",
	"/gopher/bumper.png",
	"/gopher/bumper192x108.png",
	"/gopher/bumper320x180.png",
	"/gopher/bumper480x270.png",
	"/gopher/bumper640x360.png",
	"/gopher/doc.png",
	"/gopher/frontpage.png",
	"/gopher/gopherbw.png",
	"/gopher/gophercolor.png",
	"/gopher/gophercolor16x16.png",
	"/gopher/help.png",
	"/gopher/pkg.png",
	"/gopher/project.png",
	"/gopher/ref.png",
	"/gopher/run.png",
	"/gopher/talks.png",
	"/gopher/pencil/",
	"/gopher/pencil/gopherhat.jpg",
	"/gopher/pencil/gopherhelmet.jpg",
	"/gopher/pencil/gophermega.jpg",
	"/gopher/pencil/gopherrunning.jpg",
	"/gopher/pencil/gopherswim.jpg",
	"/gopher/pencil/gopherswrench.jpg",
	"/play/",
	"/play/fib.go",
	"/play/hello.go",
	"/play/life.go",
	"/play/peano.go",
	"/play/pi.go",
	"/play/sieve.go",
	"/play/solitaire.go",
	"/play/tree.go",
	"/progs/",
	"/progs/cgo1.go",
	"/progs/cgo2.go",
	"/progs/cgo3.go",
	"/progs/cgo4.go",
	"/progs/defer.go",
	"/progs/defer.out",
	"/progs/defer2.go",
	"/progs/defer2.out",
	"/progs/eff_bytesize.go",
	"/progs/eff_bytesize.out",
	"/progs/eff_qr.go",
	"/progs/eff_sequence.go",
	"/progs/eff_sequence.out",
	"/progs/eff_unused1.go",
	"/progs/eff_unused2.go",
	"/progs/error.go",
	"/progs/error2.go",
	"/progs/error3.go",
	"/progs/error4.go",
	"/progs/go1.go",
	"/progs/gobs1.go",
	"/progs/gobs2.go",
	"/progs/image_draw.go",
	"/progs/image_package1.go",
	"/progs/image_package1.out",
	"/progs/image_package2.go",
	"/progs/image_package2.out",
	"/progs/image_package3.go",
	"/progs/image_package3.out",
	"/progs/image_package4.go",
	"/progs/image_package4.out",
	"/progs/image_package5.go",
	"/progs/image_package5.out",
	"/progs/image_package6.go",
	"/progs/image_package6.out",
	"/progs/interface.go",
	"/progs/interface2.go",
	"/progs/interface2.out",
	"/progs/json1.go",
	"/progs/json2.go",
	"/progs/json2.out",
	"/progs/json3.go",
	"/progs/json4.go",
	"/progs/json5.go",
	"/progs/run",
	"/progs/slices.go",
	"/progs/timeout1.go",
	"/progs/timeout2.go",
	"/progs/update.bash",
}

func testTrie(t *testing.T, r *Trie, method func(string) *Trie, routes, urls []string) {

	for _, s := range routes {
		trie := method(s)
		if trie.Word != nil {
			t.Fatal(s, trie.Word.(string))
		}

		s1 := trie.String()
		if s1 != s {
			t.Fatal(s, s1)
		}
		trie.Word = s
	}

	for i, s := range urls {
		m, p, err := r.Match(s, nil)
		if m == nil {
			t.Fatal(routes[i], s)
		}
		if m.Word == nil {
			t.Fatal(m.String(), p, err, routes[i], s)
		}

		if m.Word.(string) != routes[i] {
			t.Fatal(routes[i], s, m.String())
		}
	}
}

func TestTrie_Mix(t *testing.T) {
	r := newTrie('/')
	testTrie(t, r, r.Mix, routes, urls)

	r = newTrie('/')
	testTrie(t, r, r.Mix, staticRoutes, staticRoutes)
}

func TestTrie_Add(t *testing.T) {
	r := newTrie('/')
	testTrie(t, r, r.Add, routes, urls)

	r = newTrie('/')
	testTrie(t, r, r.Add, staticRoutes, staticRoutes)

	r = newTrie('.')
	testTrie(t, r, r.Add, hostRoutes, hosts)
}

func TestTrie_Params(t *testing.T) {
	r := newTrie('/')
	r.Mix("/:a/:b/:c/:d/:e/:f/:g/:h/:i/:j/:k/:l/:m/:n/:o/:p/:q/:r/:s/:t").Word = 0
	n, params, _ := r.Match("/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t", nil)
	if n == nil || params == nil || len(params) != 20 {
		r.Print()
		t.Fatal(n, "\n", params)
	}
}

func rivetHandler(c *Context) {}

func BenchmarkRivet_Static(b *testing.B) {
	r := New()

	for _, path := range staticRoutes {
		r.Get(path, rivetHandler)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, path := range staticRoutes {
			r.Match("GET", path, nil)
		}
	}
}

func BenchmarkTrie_Static(b *testing.B) {
	r := newTrie('/')

	for _, path := range staticRoutes {
		r.Mix(path).Word = 0
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, path := range staticRoutes {
			r.Match(path, nil)
		}
	}
}

func BenchmarkTrie_Params(b *testing.B) {
	r := newTrie('/')

	r.Mix("/:a/:b/:c/:d/:e/:f/:g/:h/:i/:j/:k/:l/:m/:n/:o/:p/:q/:r/:s/:t").Word = 0
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Match("/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t", nil)
	}
}
