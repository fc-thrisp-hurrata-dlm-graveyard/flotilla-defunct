package flotilla

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type (
	// Information about a route as a unit outside of the router for use & reuse
	Route struct {
		static      bool
		method      string
		path        string
		staticpath  string
		visiblepath string
		handlers    []HandlerFunc
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
)

func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	for _, handler := range middlewares {
		if !group.handlerExists(handler) {
			group.Handlers = append(group.Handlers, handler)
		}
	}
}

func (group *RouterGroup) handlerExists(outside HandlerFunc) bool {
	for _, inside := range group.Handlers {
		if funcEqual(inside, outside) {
			return true
		}
	}
	return false
}

// Creates a new router group.
func (group *RouterGroup) NewGroup(component string, handlers ...HandlerFunc) *RouterGroup {
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
	handlers := group.combineHandlers(route.handlers)
	route.visiblepath = group.pathFor(route.path)
	group.routes = append(group.routes, route)
	group.engine.router.Handle(route.method, route.visiblepath, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		c := group.engine.getCtx(w, req, params, handlers)
		//fmt.Printf("\nPRE HOOK\n")
		c.Next()
		//fmt.Printf("\nPOST HOOK\n")
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

func (group *RouterGroup) Static(staticpath string) {
	group.engine.AddStaticDir(staticpath)
	staticpath = group.pathNoLeadingSlash(staticpath)
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

// Creates a Route that is not a static route
func CommonRoute(method string, path string, handlers []HandlerFunc) *Route {
	return &Route{method: method,
		path:     path,
		handlers: handlers}
}

// Creates a Route for a static route
func StaticRoute(method string, staticpath string, handlers []HandlerFunc) *Route {
	path := filepath.Join(staticpath, "/*filepath")
	return &Route{static: true,
		method:     method,
		path:       path,
		staticpath: staticpath,
		handlers:   handlers}
}
