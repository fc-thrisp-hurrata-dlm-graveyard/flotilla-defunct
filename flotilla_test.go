package flotilla

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"

	"testing"
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

func testRouteOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testRouteOK")
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

func testGroupOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testGroupOK")
	f.Handle(NewRoute(method, "/test_group", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Configure(f.Configuration...)

	w := PerformRequest(f, method, "/test_group")

	if passed == false {
		t.Errorf(method + " group route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestGroupOK(t *testing.T) {
	for _, m := range METHODS {
		testGroupOK(m, t)
	}
}

func testSubGroupOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testsubgroupOK")
	g := f.New("/test_group")
	g.Handle(NewRoute(method, "/test_group_subgroup", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Configure(f.Configuration...)

	w := PerformRequest(f, method, "/test_group/test_group_subgroup")

	if passed == false {
		t.Errorf(method + " group route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestSubGroupOK(t *testing.T) {
	for _, m := range METHODS {
		testSubGroupOK(m, t)
	}
}

func testRouteNotOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testroutenotok")
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

	f := New("flotilla_test_Base")

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
