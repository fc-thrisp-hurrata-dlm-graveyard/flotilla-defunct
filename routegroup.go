package flotilla

import (
	"path/filepath"
	"strings"

	"github.com/thrisp/engine"
)

type (
	// A RouteGroup gathers any number routes around a prefix and an array of
	// group specific handlers.
	RouteGroup struct {
		app      *App
		prefix   string
		children []*RouteGroup
		routes   Routes
		group    *engine.Group
		Handlers []HandlerFunc
	}
)

func (rg *RouteGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(rg.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, rg.Handlers...)
	h = append(h, handlers...)
	return h
}

func (rg *RouteGroup) handlerExists(outside HandlerFunc) bool {
	for _, inside := range rg.Handlers {
		if funcEqual(inside, outside) {
			return true
		}
	}
	return false
}

func (rg *RouteGroup) pathFor(path string) string {
	joined := filepath.ToSlash(filepath.Join(rg.prefix, path))
	// Append a '/' if the last component had one, but only if it's not there already
	if len(path) > 0 && path[len(path)-1] == '/' && joined[len(joined)-1] != '/' {
		return joined + "/"
	}
	return joined
}

func (rg *RouteGroup) pathNoLeadingSlash(path string) string {
	return strings.TrimLeft(strings.Join([]string{rg.prefix, path}, "/"), "/")
}

// NewRouteGroup returns a new RouteGroup associated with the App, with the
// provided string prefix.
func NewRouteGroup(prefix string, app *App) *RouteGroup {
	return &RouteGroup{prefix: prefix,
		app:    app,
		group:  app.engine.Group.New(prefix),
		routes: make(Routes),
	}
}

// New Creates a new child RouteGroup with the base component string & handlers.
func (rg *RouteGroup) New(component string, handlers ...HandlerFunc) *RouteGroup {
	prefix := rg.pathFor(component)

	newrg := NewRouteGroup(prefix, rg.app)
	newrg.Handlers = rg.combineHandlers(handlers)

	rg.children = append(rg.children, newrg)

	return newrg
}

// Use adds any number of HandlerFunc to the RouteGroup which will be run before
// route handlers for all Route attached to the RouteGroup.
func (rg *RouteGroup) Use(middlewares ...HandlerFunc) {
	for _, handler := range middlewares {
		if !rg.handlerExists(handler) {
			rg.Handlers = append(rg.Handlers, handler)
		}
	}
}

// UseAt adds any number of HandlerFunc to the RouteGroup as middleware when you
// must control the position in relation to other middleware.
func (rg *RouteGroup) UseAt(index int, middlewares ...HandlerFunc) {
	if index > len(rg.Handlers) {
		rg.Use(middlewares...)
		return
	}

	var newh []HandlerFunc

	for _, handler := range middlewares {
		if !rg.handlerExists(handler) {
			newh = append(newh, handler)
		}
	}

	before := rg.Handlers[:index]
	after := append(newh, rg.Handlers[index:]...)
	rg.Handlers = append(before, after...)
}

func (rg *RouteGroup) addRoute(r *Route) {
	if r.Name != "" {
		rg.routes[r.Name] = r
	} else {
		rg.routes[r.Named()] = r
	}
}

// Handle registers new handlers and/or middlewares with a constructed Route.
// method. For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used by specifying path & handlers.
func (rg *RouteGroup) Handle(route *Route) {
	// finalize Route with RouteGroup specific information
	route.routergroup = rg
	route.handlers = rg.combineHandlers(route.handlers)
	route.path = rg.pathFor(route.base)
	route.p.New = route.newCtx
	rg.addRoute(route)
	// pass to engine group, using route base to register handle with the engine
	rg.group.Handle(route.base, route.method, route.handle)
}

func (rg *RouteGroup) POST(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("POST", path, false, handlers))
}

func (rg *RouteGroup) GET(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("GET", path, false, handlers))
}

func (rg *RouteGroup) DELETE(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("DELETE", path, false, handlers))
}

func (rg *RouteGroup) PATCH(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("PATCH", path, false, handlers))
}

func (rg *RouteGroup) PUT(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("PUT", path, false, handlers))
}

func (rg *RouteGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("OPTIONS", path, false, handlers))
}

func (rg *RouteGroup) HEAD(path string, handlers ...HandlerFunc) {
	rg.Handle(NewRoute("HEAD", path, false, handlers))
}

// STATIC adds a Static route handled by the app engine, based on the group prefix.
func (rg *RouteGroup) STATIC(path string) {
	rg.app.AddStaticDir(pathDropFilepathSplat(path))
	rg.Handle(NewRoute("GET", path, true, []HandlerFunc{handleStatic}))
}

// Custom HttpStatus for the group, set and called from engine HttpStatuses
func (rg *RouteGroup) StatusHandle(code int, handlers ...HandlerFunc) {
	StatusHandler := func(c *engine.Ctx) {
		statusCtx := rg.app.tmpCtx(c.RW, c.Request)
		s := len(handlers)
		for i := 0; i < s; i++ {
			handlers[i](statusCtx)
		}
	}
	if ss, ok := rg.group.HttpStatuses[code]; ok {
		ss.Update(StatusHandler)
	} else {
		ns := engine.NewHttpStatus(code, string(code))
		ns.Update(StatusHandler)
		rg.group.HttpStatuses.New(ns)
	}
}
