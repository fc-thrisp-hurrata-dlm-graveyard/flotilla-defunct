package flotilla

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
)

type (
	HandlerFunc func(*Context)

	// Information about a route as a unit outside of the router, for use & reuse
	Route struct {
		static     bool
		method     string
		path       string
		staticpath string
		handlers   []HandlerFunc
	}

	// A RouterGroup is associated with a prefix and an array of handlers
	RouterGroup struct {
		Handlers []HandlerFunc
		prefix   string
		parent   *RouterGroup
		children []*RouterGroup
		routes   []*Route
		engine   *Engine
	}

	// Basic struct that represents the web framework
	Engine struct {
		Name string
		*Env
		*RouterGroup
		cache        sync.Pool
		finalNoRoute []HandlerFunc
		noRoute      []HandlerFunc
		router       *httprouter.Router
		flotilla     map[string]Flotilla
		Assets
	}

	Flotilla interface {
		EngineName() string
		Groups() []*RouterGroup
		HasAssets() Assets
	}
)

// Returns a new blank Engine
func New(name string) *Engine {
	engine := &Engine{Name: name}
	engine.Env = BaseEnv()
	engine.RouterGroup = &RouterGroup{prefix: "/", engine: engine}
	engine.router = httprouter.New()
	engine.router.NotFound = engine.handle404
	engine.cache.New = func() interface{} {
		c := &Context{Engine: engine}
		c.Writer = &c.writermem
		return c
	}
	engine.flotilla = make(map[string]Flotilla)
	return engine
}

// Returns a basic Engine instance with sensible defaults
func Basic() *Engine {
	engine := New("flotilla")
	engine.Use(Recovery(), Logger())
	engine.Static("static")
	return engine
}

func (engine *Engine) handle404(w http.ResponseWriter, req *http.Request) {
	c := engine.createContext(w, req, nil, engine.finalNoRoute)
	c.Writer.setStatus(404)
	c.Next()
	if !c.Writer.Written() {
		c.Data(404, "text/plain", []byte("404 page not found"))
	}
	engine.cache.Put(c)
}

//merge other engine(routes, handlers, middleware, etc) with existing engine
func (engine *Engine) Extend(f Flotilla) error {
	engine.flotilla[f.EngineName()] = f
	fmt.Printf("extending with: %+v\n", f)
	//for _, x := range f.Groups() {
	//	fmt.Printf("group: %+v\n", x)
	//}
	for _, fs := range f.HasAssets() {
		engine.Assets = append(engine.Assets, fs)
	}
	return nil
}

// Adds handlers for NoRoute
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.finalNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) Use(middlewares ...HandlerFunc) {
	engine.RouterGroup.Use(middlewares...)
	engine.finalNoRoute = engine.combineHandlers(engine.noRoute)
}

// ServeHTTP makes the router implement the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

func (engine *Engine) Run(addr string) {
	if err := http.ListenAndServe(addr, engine); err != nil {
		panic(err)
	}
}

//methods to ensure *Engine satisfies interface Flotilla
func (engine *Engine) EngineName() string {
	return engine.Name
}

func (engine *Engine) HasAssets() Assets {
	if engine.Assets != nil {
		return engine.Assets
	}
	return nil
}

func (engine *Engine) Groups() []*RouterGroup {
	type IterC func(r []*RouterGroup, fn IterC)

	var rg []*RouterGroup

	rg = append(rg, engine.RouterGroup)

	iter := func(r []*RouterGroup, fn IterC) {
		for _, x := range r {
			rg = append(rg, x)
			fn(x.children, fn)
		}
	}

	iter(engine.children, iter)

	return rg
}

// list of flotilla, starting with current
func (engine *Engine) Flotilla() []Flotilla {
	var ret []Flotilla
	ret = append(ret, engine)
	for _, e := range engine.flotilla {
		ret = append(ret, e)
	}
	return ret
}

// ROUTES GROUPING //
// Adds handler middlewares to the group.
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// Creates a new router group.
func (group *RouterGroup) Group(component string, handlers ...HandlerFunc) *RouterGroup {
	prefix := group.pathFor(component)

	newroutergroup := &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		parent:   group,
		prefix:   prefix,
		engine:   group.engine,
	}

	group.children = append(group.children, newroutergroup)

	return newroutergroup
}

func (group *RouterGroup) pathFor(path string) string {
	joined := filepath.Join(group.prefix, path)
	// Append a '/' if the last component had one, but only if it's not there already
	if len(path) > 0 && path[len(path)-1] == '/' && joined[len(joined)-1] != '/' {
		return joined + "/"
	}
	return joined
}

//a non-absolute path fragment for the group provided a path
func (group *RouterGroup) pathNoLeadingSlash(path string) string {
	return strings.TrimLeft(strings.Join([]string{group.prefix, path}, "/"), "/")
}

// Handle registers a new request handle and middlewares with the given path and method.
// The last handler should be the real handler, the other ones should be middlewares that can and should be shared among different routes.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (group *RouterGroup) Handle(route *Route) {
	path := group.pathFor(route.path)
	handlers := group.combineHandlers(route.handlers)
	group.routes = append(group.routes, route)
	group.engine.router.Handle(route.method, path, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		c := group.engine.createContext(w, req, params, handlers)
		c.Next()
		group.engine.cache.Put(c)
	})
}

func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("POST", path, handlers))
}

func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("GET", path, handlers))
}

func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("DELETE", path, handlers))
}

func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("PATCH", path, handlers))
}

func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("PUT", path, handlers))
}

func (group *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("OPTIONS", path, handlers))
}

func (group *RouterGroup) HEAD(path string, handlers ...HandlerFunc) {
	group.Handle(CommonRoute("HEAD", path, handlers))
}

// Static serves files from the given file system root.
func (group *RouterGroup) Static(staticpath string) {
	staticpath = group.pathNoLeadingSlash(staticpath)
	group.engine.AddStaticPath(staticpath)
	group.Handle(StaticRoute("GET", staticpath, []HandlerFunc{handleStatic}))
	group.Handle(StaticRoute("HEAD", staticpath, []HandlerFunc{handleStatic}))
}

func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}

func CommonRoute(method string, path string, handlers []HandlerFunc) *Route {
	return &Route{method: method, path: path, handlers: handlers}
}

func StaticRoute(method string, staticpath string, handlers []HandlerFunc) *Route {
	path := filepath.Join(staticpath, "/*filepath")
	return &Route{method: method, path: path, static: true, staticpath: staticpath, handlers: handlers}
}
