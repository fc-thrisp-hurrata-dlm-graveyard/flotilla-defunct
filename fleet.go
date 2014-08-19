package fleet

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
)

type (
	HandlerFunc func(*Context)

	// Information about a route as a unit outside of the router, for use & reuse
	Route struct {
		method   string
		path     string
		handlers []HandlerFunc
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
		*FleetEnv
		*RouterGroup
		cache        sync.Pool
		finalNoRoute []HandlerFunc
		noRoute      []HandlerFunc
		router       *httprouter.Router
		engines      []*Engine
		*AssetFS
	}
)

// Returns a new blank Engine
func New(name string) *Engine {
	engine := &Engine{Name: name}
	engine.FleetEnv = &FleetEnv{}
	engine.RouterGroup = &RouterGroup{prefix: "/", engine: engine}
	engine.router = httprouter.New()
	engine.router.NotFound = engine.handle404
	engine.cache.New = func() interface{} {
		c := &Context{Engine: engine}
		c.Writer = &c.writermem
		return c
	}
	return engine
}

// Returns a basic Engine instance with sensible defaults
func Basic() *Engine {
	engine := New("fleet")
	engine.Use(Recovery(), Logger())
	engine.FleetEnv = NewFileEnv("")
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
func (engine *Engine) Merge(e *Engine) error {
	engine.engines = append(engine.engines, e)
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

//func (group *RouterGroup) newRoute(method string, path string, handlers ...HandlerFunc) *Route {
//	return &Route{method, path, handlers}
//}

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
	group.Handle(&Route{"POST", path, handlers})
}

func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"GET", path, handlers})
}

func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"DELETE", path, handlers})
}

func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"PATCH", path, handlers})
}

func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"PUT", path, handlers})
}

func (group *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"OPTIONS", path, handlers})
}

func (group *RouterGroup) HEAD(path string, handlers ...HandlerFunc) {
	group.Handle(&Route{"HEAD", path, handlers})
}

// Static serves files from the given file system root.
func (group *RouterGroup) Static(staticpath string) {
	group.engine.AddStaticPath(group.pathNoLeadingSlash(staticpath))
	staticpath = filepath.Join(staticpath, "/*filepath")
	group.Handle(&Route{"GET", staticpath, []HandlerFunc{group.handleStatic}})
	group.Handle(&Route{"HEAD", staticpath, []HandlerFunc{group.handleStatic}})
}

func (group *RouterGroup) handleStatic(c *Context) {
	requested := filepath.Base(c.Request.URL.Path)
	// check current paths
	for _, dir := range c.Engine.StaticPaths {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) error {
			if filepath.Base(path) == requested {
				c.File(path)
			}
			return nil
		})
	}
	// check main engine AssetFS, if it exists
	if c.Engine.AssetFS != nil {
		if hasfile, ok := c.Engine.HasAsset(requested); ok {
			c.ServeAssetFS(hasfile, c.Engine.AssetFS)
		}
	}
	// check each engine AssetFS
	if c.Engine.engines != nil {
		for _, engine := range c.Engine.engines {
			if engine.AssetFS != nil {
				if hasfile, ok := engine.HasAsset(requested); ok {
					c.ServeAssetFS(hasfile, engine.AssetFS)
				}
			}
		}
	}
	//not found
}

func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}
