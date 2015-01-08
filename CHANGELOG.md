##Changelog

### Flotilla 0.3.2 ()

- change 'ctx functions' to 'extensions'

### Flotilla 0.3.1 (12.16.2014)

- update travis.yml to accomodate 1.4/1.3 cover package path difference

### Flotilla 0.3.0 (12.15.2014)

- new Blueprint concepts 
- eliminate old Blueprint interface, merge RouteGroup & Blueprint idioms to one
- engine interface & default engine, for future extensible engines
- essential testing, bugfixes, and refactoring

### Flotilla 0.2.0 (12.2.2014) ~unreleased

- return from 'R' to 'Ctx'
- flash messaging
- per-route, in-template context processors
- initialization & configuration streamlining
- coinciding boolean app modes (development, testing, & production)
- methods for viewing & setting plain or minimally secure cookies
- public store item value for easier access to store settings 
- deferral of ctx functions until after all handlers have run
- bugfixes & refactoring  

### Flotilla 0.1.0 (10.17.2014)

- basic adherence to semantic versioning
- 'R' type for per-route handled context
- reintegrate Djinn(formerly Jingo) templating
- tighter interaction with Engine statuses & panics
- simple configuration functions, removal of flags 
- package-level errors
- essential testing, bugfixes, and refactoring  


### Flotilla 0.0.2 (9.24.2014)

- extend Ctx with cross handler functions
- simple flag parsing for run mode (production, development, testing)
- cookie based sessions as a default with capacity for adding different backends
- folded router & some lower-level-but-not-in-net/http(such as http statuses)
  functions into another package: [Engine](https://github.com/thrisp/engine)
- url formatting by route, i.e. creating urls by route name & parameters


### Flotilla 0.0.1 (8.20.2014)

- reforked, renamed as Flotilla
- ini style configuration read into app environment
- basic Flotilla interface for extension of routes & env
- basic Jingo templating
- provisions for binary static or template Asset inclusion per engine
 

### Fleet 0.0.0 (7.22.2014)

- forked from https://github.com/gin-gonic/gin
