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
		p           sync.Pool
		routergroup *RouteGroup
		static      bool
		method      string
		base        string
		path        string
		handlers    []HandlerFunc
		ctxprcss    map[string]interface{}
		Name        string
	}

	// A map of Route instances keyed by a string.
	Routes map[string]*Route
)

func (rt *Route) handle(ec *engine.Ctx) {
	rq := rt.getCtx(ec)
	rq.Next()
	rt.putR(rq)
}

// NewRoute returns a new Route from a string method, a string path, a boolean
// indicating if the route is static, and an aray of HandlerFunc
func NewRoute(method string, path string, static bool, handlers []HandlerFunc) *Route {
	rt := &Route{method: method, static: static, handlers: handlers, ctxprcss: make(map[string]interface{})}
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

// Named produces a default name for the route based on path & parameters, useful
// to RouteGroup and App, where a route is not specifically named.
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

// Url takes string parameters and applies them to a Route. First to any :parameter
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

func (rt *Route) CtxProcessor(name string, fn interface{}) {
	rt.ctxprcss[name] = fn
}

func (rt *Route) CtxProcessors(cp map[string]interface{}) {
	for k, v := range cp {
		rt.CtxProcessor(k, v)
	}
}
