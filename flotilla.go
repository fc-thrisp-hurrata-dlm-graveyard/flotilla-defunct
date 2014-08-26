package flotilla

import (
	"fmt"
	"net/http"

	"sync"

	"github.com/julienschmidt/httprouter"
)

type (
	HandlerFunc func(*Context)

	Engine struct {
		Name string
		*Env
		*RouterGroup
		cache        sync.Pool
		finalNoRoute []HandlerFunc
		noRoute      []HandlerFunc
		router       *httprouter.Router
		flotilla     map[string]Flotilla
	}

	Blueprint struct {
		Name   string
		Prefix string
		Groups []*RouterGroup
		Env    *Env
	}

	Flotilla interface {
		Blueprint() *Blueprint
	}
)

// Returns a new, default Engine
func New(name string) *Engine {
	engine := &Engine{}
	engine.Name = name
	engine.Env = BaseEnv()
	engine.RouterGroup = &RouterGroup{prefix: "/", engine: engine}
	engine.router = httprouter.New()
	engine.router.NotFound = engine.default404
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

//merge other engine(routes, handlers, middleware, etc) with existing engine
func (engine *Engine) Extend(f Flotilla) error {
	fmt.Printf("extending with flotilla %+v", f)
	b := f.Blueprint()
	//name
	engine.flotilla[b.Name] = f
	//groups
	for _, x := range b.Groups {
		if _, ok := engine.existingGroup(x.prefix); ok {
			fmt.Printf("\ngroup with %s matches EXISTING group\n", x.prefix)
			fmt.Printf("\n%+v\n", x)
		} else {
			fmt.Printf("\ngroup with %s is a NEW group\n", x.prefix)
			fmt.Printf("\n%+v\n", x)
		}
		for _, y := range x.routes {
			fmt.Printf("\nroute: %+v\n", y)
		}
	}
	//conf
	fmt.Printf("\nblueprint *Env %+v\n", b.Env)
	engine.LoadConfMap(b.Env.Conf)
	//assets
	for _, fs := range b.Env.Assets {
		engine.Env.Assets = append(engine.Env.Assets, fs)
	}
	//
	fmt.Printf("\n*Env.Templator %+v\n", b.Env.Templator)
	for _, l := range b.Env.Templator.Loaders {
		fmt.Printf("\n*Env.Templator has loader %+v\n", l)
	}
	return nil
}

func (engine *Engine) default404(w http.ResponseWriter, req *http.Request) {
	c := engine.createContext(w, req, nil, engine.finalNoRoute)
	c.Writer.WriteHeader(404)
	c.Next()
	if !c.Writer.Written() {
		if c.Writer.Status() == 404 {
			c.Data(-1, "text/plain", []byte("404 page not found"))
		} else {
			c.Writer.WriteHeaderNow()
		}
	}
	engine.cache.Put(c)
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
func (engine *Engine) Blueprint() *Blueprint {
	return &Blueprint{Name: engine.Name,
		Groups: engine.Groups(),
		Env:    engine.Env}
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

// list of flotilla, starting with calling *Engine
func (engine *Engine) Flotilla() []Flotilla {
	var ret []Flotilla
	ret = append(ret, engine)
	for _, e := range engine.flotilla {
		ret = append(ret, e)
	}
	return ret
}

func (engine *Engine) existingGroup(prefix string) (*RouterGroup, bool) {
	for _, g := range engine.Groups() {
		if g.prefix == prefix {
			return g, true
		}
	}
	return nil, false
}
