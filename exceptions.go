package flotilla

/*
// A handler for httprouter NotFound handler
func (engine *Engine) handler404(w http.ResponseWriter, req *http.Request) {
	e := engine.HttpExceptions[404]
	c := engine.getCtx(w, req, nil, engine.combineHandlers(e.handlers))
	c.Next()
	engine.cache.Put(c)
}

// A handler for httprouter Panic handler
func (engine *Engine) handler500(w http.ResponseWriter, req *http.Request, err interface{}) {
	e := engine.HttpExceptions[500]
	e.updatehandlers(func(c *Ctx) {
		stack := stack(3)
		log.Printf("\n-----\nPANIC\n-----\nerr: %s\n-----\n%s\n-----\n", err, stack)
		switch engine.Env.Mode {
		case devmode, testmode:
			servePanic := fmt.Sprintf(panicHtml, err, err, stack)
			c.ServeData(500, "text/html", []byte(servePanic))
		}
	})
	c := engine.getCtx(w, req, nil, engine.combineHandlers(e.handlers))
	c.Next()
	engine.cache.Put(c)
}
*/
