##Changelog

### Flotilla 0.1.1 (tbd)

- flash messaging


### Flotilla 0.1.0 (10.17.2014)

- semantic versioning
- 'R' type for per-route handled context
- reintegrate Djinn(formerly Jingo) templating
- tighter interaction of Engine statuses & panics
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
