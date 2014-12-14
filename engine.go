package flotilla

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/thrisp/engine"
	"golang.org/x/net/context"
)

type (
	SetEngine func(*App) error

	Engine interface {
		Take(string, string, func(context.Context))
		TakeStatus(int, func(context.Context))
		Reconfigure(func() error) error
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
)

func DefaultEngine(a *App) error {
	a.Engine = defaultEngine()
	a.deferred = append(a.deferred, reconfigureDefault)
	return nil
}

func defaultEngine() *engine.Engine {
	e, err := engine.New(engine.HTMLStatus(true))
	if err != nil {
		panic(fmt.Sprintf("[FLOTILLA] engine could not be created properly: %s", err))
	}
	return e
}

func reconfigureDefault(a *App) error {
	re := func() error {
		e := a.Engine.(*engine.Engine)
		var cnf []engine.Conf
		if mm, err := a.Env.Store["UPLOAD_SIZE"].Int64(); err == nil {
			cnf = append(cnf, engine.MaxFormMemory(mm))
		}
		if a.Mode.Production {
			cnf = append(cnf, engine.ServePanic(false))
		}
		if !a.Mode.Production {
			cnf = append(cnf, engine.Logger(log.New(os.Stdout, "[FLOTILLA]", 0)))
		}
		if err := e.SetConf(cnf...); err != nil {
			return err
		}
		return nil
	}

	if err := a.Engine.Reconfigure(re); err != nil {
		return err
	}

	return nil
}
