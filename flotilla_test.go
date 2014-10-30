package flotilla

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"

	"testing"
)

func PerformRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func methodNotMethod(method string) string {
	methods := []string{"GET", "POST", "PATCH", "DELETE", "PUT", "OPTIONS", "HEAD"}
	newmethod := methods[rand.Intn(len(methods))]
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
	f.Env.SessionInit()

	w := PerformRequest(f, method, "/test")

	if passed == false {
		t.Errorf(method + " route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestRouteOK(t *testing.T) {
	testRouteOK("POST", t)
	testRouteOK("DELETE", t)
	testRouteOK("PATCH", t)
	testRouteOK("PUT", t)
	testRouteOK("OPTIONS", t)
	testRouteOK("HEAD", t)
}

func testGroupOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testGroupOK")
	f.Handle(NewRoute(method, "/test_group", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Env.SessionInit()

	w := PerformRequest(f, method, "/test_group")

	if passed == false {
		t.Errorf(method + " group route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestGroupOK(t *testing.T) {
	testRouteOK("POST", t)
	testRouteOK("DELETE", t)
	testRouteOK("PATCH", t)
	testRouteOK("PUT", t)
	testRouteOK("OPTIONS", t)
	testRouteOK("HEAD", t)
}

func testSubGroupOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testsubgroupOK")
	g := f.New("/test_group")
	g.Handle(NewRoute(method, "/test_group_subgroup", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Env.SessionInit()

	w := PerformRequest(f, method, "/test_group/test_group_subgroup")

	if passed == false {
		t.Errorf(method + " group route handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestSubGroupOK(t *testing.T) {
	testSubGroupOK("POST", t)
	testSubGroupOK("DELETE", t)
	testSubGroupOK("PATCH", t)
	testSubGroupOK("PUT", t)
	testSubGroupOK("OPTIONS", t)
	testSubGroupOK("HEAD", t)
}

func testRouteNotOK(method string, t *testing.T) {
	passed := false
	f := New("flotilla_testroutenotok")
	othermethod := methodNotMethod(method)
	f.Handle(NewRoute(othermethod, "/test", false, []HandlerFunc{func(ctx *Ctx) { passed = true }}))
	f.Env.SessionInit()

	w := PerformRequest(f, method, "/test")

	if passed == true {
		t.Errorf(method + " route handler was invoked, when it should not")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("Status code should be %v, was %d. Location: %s", http.StatusNotFound, w.Code, w.HeaderMap.Get("Location"))
	}
}

func TestRouteNotOK(t *testing.T) {
	testRouteNotOK("POST", t)
	testRouteNotOK("DELETE", t)
	testRouteNotOK("PATCH", t)
	testRouteNotOK("PUT", t)
	testRouteNotOK("OPTIONS", t)
	testRouteNotOK("HEAD", t)
}

/*func TestHandleStaticFile(t *testing.T) {
	// SETUP file
	testRoot, _ := os.Getwd()
	f, err := ioutil.TempFile(testRoot, "")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(f.Name())
	filePath := path.Join("/", path.Base(f.Name()))
	f.WriteString("Flotilla Web Framework")
	f.Close()

	// SETUP
	r := New("flotilla_test_testhandlestaticfile")
	r.Static("/")

	// RUN
	w := PerformRequest(r, "GET", filePath)

	// TEST
	if w.Code != 200 {
		t.Errorf("Response code should be Ok, was: %s", w.Code)
	}
	if w.Body.String() != "Flotilla Web Framework" {
		t.Errorf("Response should be test, was: %s", w.Body.String())
	}
	if w.HeaderMap.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type should be text/plain, was %s", w.HeaderMap.Get("Content-Type"))
	}
}*/

//func TestHandleStaticBinaryFile(t *testing.T) {}

/*func TestHandleStaticDir(t *testing.T) {
	// SETUP
	r := New()
	r.Static("/", "./")

	// RUN
	w := PerformRequest(r, "GET", "/")

	// TEST
	bodyAsString := w.Body.String()
	if w.Code != 200 {
		t.Errorf("Response code should be Ok, was: %s", w.Code)
	}
	if len(bodyAsString) == 0 {
		t.Errorf("Got empty body instead of file tree")
	}
	if !strings.Contains(bodyAsString, "gin.go") {
		t.Errorf("Can't find:`gin.go` in file tree: %s", bodyAsString)
	}
	if w.HeaderMap.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Content-Type should be text/plain, was %s", w.HeaderMap.Get("Content-Type"))
	}
}*/

/*func TestHandleHeadToDir(t *testing.T) {
	// SETUP
	r := New("flotilla_test_testhandleheadtodir")
	r.Static("/teststaticheadtodir")

	// RUN
	w := PerformRequest(r, "HEAD", "/teststaticheadtodir/")

	// TEST
	bodyAsString := w.Body.String()
	if w.Code != 200 {
		t.Errorf("Response code should be Ok, was: %s", w.Code)
	}
	if len(bodyAsString) == 0 {
		t.Errorf("Got empty body instead of file tree")
	}
	if !strings.Contains(bodyAsString, "flotilla.go") {
		t.Errorf("Can't find:`flotilla.go` in file tree: %s", bodyAsString)
	}
	if w.HeaderMap.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Content-Type should be text/plain, was %s", w.HeaderMap.Get("Content-Type"))
	}
}*/

// basic test of extension existence
func TestExtension(t *testing.T) {
	r := New("flotilla_test_testExtension_base")
	r1 := New("flotilla_test_testExtension_extension")

	r.Extend(r1)

	if ext, ok := r.flotilla[r1.Name]; !ok {
		t.Errorf("%s:%v basic extension was not found in %s:%v", r1.Name, ext, r.Name, r)
	}
}

// Tests that an engine route is correctly extended
func testExtensionRouteOK(method string, t *testing.T) {
	passed := false
	r := New(fmt.Sprintf("flotilla_test_testExtensionRouteOK_base_%s", method))
	r1 := New(fmt.Sprintf("flotilla_test_testExtensionRouteOK_extension_%s", method))
	rt := NewRoute(method, "/extension_test", false, []HandlerFunc{func(ctx *Ctx) {
		passed = true
	}})
	r1.Handle(rt)
	r.Extend(r1)
	r.Env.SessionInit()

	w := PerformRequest(r, method, "/extension_test")

	if passed == false {
		t.Errorf(method + " extended handler was not invoked.")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d", http.StatusOK, w.Code)
	}
}

func TestExtensionRouteOK(t *testing.T) {
	testExtensionRouteOK("POST", t)
	testExtensionRouteOK("DELETE", t)
	testExtensionRouteOK("PATCH", t)
	testExtensionRouteOK("PUT", t)
	testExtensionRouteOK("OPTIONS", t)
	testExtensionRouteOK("HEAD", t)
}
