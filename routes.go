package flotilla

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"

	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	regParam = regexp.MustCompile(`:[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)
	regSplat = regexp.MustCompile(`\*[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)
)

type (
	// Information about a route as a unit outside of the router for use & reuse.
	Route struct {
		Name     string
		static   bool
		method   string
		base     string
		path     string
		handlers []HandlerFunc
	}

	// A map of Route instances keyed by a string
	Routes map[string]*Route

	// A RouterGroup is associated with a prefix and an array of handlers.
	RouterGroup struct {
		Handlers []HandlerFunc
		prefix   string
		parent   *RouterGroup
		children []*RouterGroup
		routes   Routes
		engine   *Engine
	}
)

func NewRoute(method string, path string, static bool, handlers []HandlerFunc) *Route {
	r := &Route{method: method, static: static, handlers: handlers}
	if static {
		if fp := strings.Split(path, "/"); fp[len(fp)-1] != "*filepath" {
			r.base = filepath.Join(path, "/*filepath")
		} else {
			r.base = path
		}
	} else {
		r.base = path
	}
	return r
}

func (r *Route) Named() string {
	name := strings.Split(r.path, "/")
	name = append(name, strings.ToLower(r.method))
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
func (r *Route) Url(params ...string) (*url.URL, error) {
	paramCount := len(params)
	i := 0
	rurl := regParam.ReplaceAllStringFunc(r.path, func(m string) string {
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
	if i < len(params) && r.method == "GET" {
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

func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}

func (group *RouterGroup) handlerExists(outside HandlerFunc) bool {
	for _, inside := range group.Handlers {
		if funcEqual(inside, outside) {
			return true
		}
	}
	return false
}

func (group *RouterGroup) pathFor(path string) string {
	joined := filepath.Join(group.prefix, path)
	// Append a '/' if the last component had one, but only if it's not there already
	if len(path) > 0 && path[len(path)-1] == '/' && joined[len(joined)-1] != '/' {
		return joined + "/"
	}
	return joined
}

func (group *RouterGroup) pathNoLeadingSlash(path string) string {
	return strings.TrimLeft(strings.Join([]string{group.prefix, path}, "/"), "/")
}

func (group *RouterGroup) pathDropFilepathSplat(path string) string {
	if fp := strings.Split(path, "/"); fp[len(fp)-1] == "*filepath" {
		return strings.Join(fp[0:len(fp)-1], "/")
	}
	return path
}

func NewRouterGroup(prefix string, engine *Engine) *RouterGroup {
	return &RouterGroup{prefix: prefix,
		engine: engine,
		routes: make(Routes),
	}
}

// Creates a new router group.
func (group *RouterGroup) New(component string, handlers ...HandlerFunc) *RouterGroup {
	prefix := group.pathFor(component)

	newroutergroup := NewRouterGroup(prefix, group.engine)
	newroutergroup.parent = group
	newroutergroup.Handlers = group.combineHandlers(handlers)

	group.children = append(group.children, newroutergroup)

	return newroutergroup
}

// Adds any number of HandlerFunc to the RouterGroup as middleware.
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	for _, handler := range middlewares {
		if !group.handlerExists(handler) {
			group.Handlers = append(group.Handlers, handler)
		}
	}
}

// Adds any number of HandlerFunc to the RouterGroup as middleware when you
// must control the position, use with caution.
func (group *RouterGroup) UseAt(index int, middlewares ...HandlerFunc) {
	if index > len(group.Handlers) {
		group.Use(middlewares...)
		return
	}

	var newh []HandlerFunc

	for _, handler := range middlewares {
		if !group.handlerExists(handler) {
			newh = append(newh, handler)
		}
	}

	before := group.Handlers[:index]
	after := append(newh, group.Handlers[index:]...)
	group.Handlers = append(before, after...)
}

// Adds a Route to the group routes map, using the Route.Name if provided or
// the default route name if not.
func (group *RouterGroup) AddRoute(r *Route) {
	if r.Name != "" {
		group.routes[r.Name] = r
	} else {
		group.routes[r.Named()] = r
	}
}

// Handle registers a new request handle and middlewares with the given path and
// method. For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (group *RouterGroup) Handle(route *Route) {
	handlers := group.combineHandlers(route.handlers)
	route.path = group.pathFor(route.base)
	group.AddRoute(route)
	group.engine.router.Handle(route.method, route.path, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		c := group.engine.getCtx(w, req, params, handlers)
		c.Next()
		group.engine.cache.Put(c)
	})
}

func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("POST", path, false, handlers))
}

func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("GET", path, false, handlers))
}

func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("DELETE", path, false, handlers))
}

func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("PATCH", path, false, handlers))
}

func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("PUT", path, false, handlers))
}

func (group *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("OPTIONS", path, false, handlers))
}

func (group *RouterGroup) HEAD(path string, handlers ...HandlerFunc) {
	group.Handle(NewRoute("HEAD", path, false, handlers))
}

// Adds a Static route handled by the router, based on the group prefix.
func (group *RouterGroup) STATIC(path string) {
	group.engine.AddStaticDir(group.pathDropFilepathSplat(path))
	group.Handle(NewRoute("GET", path, true, []HandlerFunc{handleStatic}))
}
