package flotilla

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

var METHODS []string = []string{"GET", "POST", "PATCH", "DELETE", "PUT", "OPTIONS", "HEAD"}

func PerformRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func methodNotMethod(method string) string {
	newmethod := METHODS[rand.Intn(len(METHODS))]
	if newmethod == method {
		methodNotMethod(newmethod)
	}
	return newmethod
}

func TestingEngine(a *App) error {
	a.Engine = newtestengine()
	return nil
}

type TestEngine struct {
	routes map[string]func(context.Context)
}

func newtestengine() *TestEngine {
	return &TestEngine{routes: make(map[string]func(context.Context))}
}

func (te *TestEngine) Take(method string, route string, handler func(context.Context)) {
	k := fmt.Sprintf("%s:%s", method, route)
	te.routes[k] = handler
}

func (te *TestEngine) TakeStatus(code int, handler func(context.Context)) {}

func (te *TestEngine) Reconfigure(func() error) error {
	return nil
}

func (te *TestEngine) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	if rt, ok := te.routes[fmt.Sprintf("%s:%s", method, path)]; ok {
		rt(context.WithValue(context.Background(), "current", true))
	}
}

func testCustomEngine(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testRouteOK", TestingEngine)
	f.Take(method, "/test", func(ctx context.Context) {
		if ctx.Value("current").(bool) {
			passed = true
		}
	})
	PerformRequest(f, method, "/test")
	if passed == false {
		t.Errorf("test engine was not used")
	}
}

func TestCustomEngine(t *testing.T) {
	for _, m := range METHODS {
		testCustomEngine(m, t)
	}
}

func testRouteOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testRouteOK", DefaultEngine)
	r := NewRoute(method, "/test", false, []HandlerFunc{func(ctx *Ctx) { passed = true }})
	f.Handle(r)
	f.Configure(f.Configuration...)

	w := PerformRequest(f, method, "/test")

	if passed == false {
		t.Errorf(method + " route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestRouteOK(t *testing.T) {
	for _, m := range METHODS {
		testRouteOK(m, t)
	}
}

func testRouteNotOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testroutenotok", DefaultEngine)
	othermethod := methodNotMethod(method)
	f.Handle(NewRoute(othermethod, "/test_notfound", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Configure(f.Configuration...)

	w := PerformRequest(f, method, "/test_notfound")

	if passed == true {
		t.Errorf(method + " route handler was invoked, when it should not")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("Status code should be %v, was %d. Location: %s", http.StatusNotFound, w.Code, w.HeaderMap.Get("Location"))
	}
}

func TestRouteNotOK(t *testing.T) {
	for _, m := range METHODS {
		testRouteNotOK(m, t)
	}
}

func testBlueprintRoute(method string, t *testing.T) {
	passed := false

	f := New("flotilla_test_Blueprint", DefaultEngine)

	b := NewBlueprint("/blueprint")

	blueprintroute := NewRoute(method, "/test_blueprint", false, []HandlerFunc{func(ctx *Ctx) {
		passed = true
	}})

	b.Handle(blueprintroute)

	f.RegisterBlueprints(b)

	f.Configure(f.Configuration...)

	expected := "/blueprint/test_blueprint"

	w := PerformRequest(f, method, expected)

	if passed == false {
		t.Errorf(fmt.Sprintf("%s blueprint route: %s was not invoked.", method, expected))
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestBlueprintRoute(t *testing.T) {
	for _, m := range METHODS {
		testBlueprintRoute(m, t)
	}
}

func testMountBlueprint(method string, t *testing.T) {
	passed := false

	f := New("flotilla_test_BlueprintMount", DefaultEngine)

	b := NewBlueprint("/mount")

	blueprintroute := NewRoute(method, "/test_blueprint", false, []HandlerFunc{func(ctx *Ctx) {
		passed = true
	}})

	b.Handle(blueprintroute)

	f.Mount("/testone", true, b)

	f.Mount("/testtwo", false, b)

	f.RegisterBlueprints(b)

	f.Configure(f.Configuration...)

	err := f.Mount("/cannot", false, b)

	if err == nil {
		t.Errorf("mounting a registered blueprint return no error")
	}

	perform := func(expected string, method string, app *App) {
		PerformRequest(app, method, expected)

		if passed == false {
			t.Errorf(fmt.Sprintf("%s blueprint route: %s was not invoked.", method, expected))
		}

		passed = false
	}

	perform("/testone/mount/test_blueprint", method, f)
	perform("/testtwo/mount/test_blueprint", method, f)
}

func TestMountBlueprint(t *testing.T) {
	for _, m := range METHODS {
		testMountBlueprint(m, t)
	}
}
