package flotilla

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/thrisp/engine"
)

var (
	regParam = regexp.MustCompile(`:[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)
	regSplat = regexp.MustCompile(`\*[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)
)

type (
	// Data about a route for use & reuse within App.
	Route struct {
		cache       sync.Pool
		routergroup *RouteGroup
		static      bool
		method      string
		base        string
		path        string
		handlers    []HandlerFunc
		Name        string
	}

	// A map of Route instances keyed by a string.
	Routes map[string]*Route

	// A RouteGroup is data about gathering any number routes around a prefix
	// and an array of group specific handlers.
	RouteGroup struct {
		app      *App
		prefix   string
		children []*RouteGroup
		routes   Routes
		group    *engine.Group
		Handlers []HandlerFunc
		HttpStatuses
	}

	RouteGroups []*RouteGroup
)

func (rt *Route) handle(c *engine.Ctx) {
	rq := rt.getR(c)
	rq.Next()
	rt.putR(rq)
}

func NewRoute(method string, path string, static bool, handlers []HandlerFunc) *Route {
	rt := &Route{method: method, static: static, handlers: handlers}
	if static {
		if fp := strings.Split(path, "/"); fp[len(fp)-1] != "*filepath" {
			rt.base = filepath.ToSlash(filepath.Join(path, "/*filepath"))
		} else {
			rt.base = path
		}
	} else {
		rt.base = path
	}
	return rt
}

func (rt *Route) Named() string {
	name := strings.Split(rt.path, "/")
	name = append(name, strings.ToLower(rt.method))
	for index, value := range name {
		if regSplat.MatchString(value) {
			name[index] = "s"
		}
		if regParam.MatchString(value) {
			name[index] = "p"
		}
	}
	return strings.Join(name, `\`)
}

// Takes string parameters and applies them to a Route. First to any :parameter
// params, then *splat params. If any params are left over(not the case with a
// *splat), and the route method is GET, a query string of key=value is appended
// to the end of the url with arbitrary assigned keys(e.g. value1=param) where
// no key is provided
//
// e.g.
// r1 := NewRoute("GET", /my/:mysterious/path, false, [AHandlerFunc])
// r2 := NewRoute("GET", /my/*path, false, [AHandlerFunc])
// u1, _ := r1.Url("hello", "world=are" "you=there", "sayhi")
// u2, _ := r2.Url("hello", "world", "are" "you", "there")
// fmt.Printf("url1: %s\n", u1)
//
//	/my/hello/path?world=are&you=there&value3=sayhi
//
// fmt.Printf("url2: %s\n", u2)
//
//	/my/hello/world/are/you/there
func (rt *Route) Url(params ...string) (*url.URL, error) {
	paramCount := len(params)
	i := 0
	rurl := regParam.ReplaceAllStringFunc(rt.path, func(m string) string {
		var val string
		if i < paramCount {
			val = params[i]
		}
		i += 1
		return fmt.Sprintf(`%s`, val)
	})
	rurl = regSplat.ReplaceAllStringFunc(rurl, func(m string) string {
		splat := params[i:(len(params))]
		i += len(splat)
		return fmt.Sprintf(strings.Join(splat, "/"))
	})
	u, err := url.Parse(rurl)
	if err != nil {
		return nil, err
	}
	if i < len(params) && rt.method == "GET" {
		providedquerystring := params[i:(len(params))]
		var querystring []string
		qsi := 0
		for qi, qs := range providedquerystring {
			if len(strings.Split(qs, "=")) != 2 {
				qs = fmt.Sprintf("value%d=%s", qi+1, qs)
			}
			querystring = append(querystring, url.QueryEscape(qs))
			qsi += 1
		}
		u.RawQuery = strings.Join(querystring, "&")
	}
	return u, nil
}

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

// NewRouteGroup attaches a new RouteGroup to the App with the prefix.
func NewRouteGroup(prefix string, app *App) *RouteGroup {
	return &RouteGroup{prefix: prefix,
		app:          app,
		group:        app.engine.Group.New(prefix),
		routes:       make(Routes),
		HttpStatuses: make(HttpStatuses),
	}
}

// New Creates a new child router group with the base component string.
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

// Handle registers new handlers and/or middlewares with the given path and
// method. For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (rg *RouteGroup) Handle(route *Route) {
	// finalize Route with RouteGroup specific information
	route.routergroup = rg
	route.handlers = rg.combineHandlers(route.handlers)
	route.path = rg.pathFor(route.base)
	route.cache.New = route.newR
	rg.addRoute(route)
	// pass to engine group, using route base to register handle with router
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

// Adds a Static route handled by the router, based on the group prefix.
func (rg *RouteGroup) STATIC(path string) {
	rg.app.AddStaticDir(pathDropFilepathSplat(path))
	rg.Handle(NewRoute("GET", path, true, []HandlerFunc{handleStatic}))
}
