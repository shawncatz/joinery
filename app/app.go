package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var app *Application

type setupFunc func(app *Application) error
type healthFunc func(app *Application) error

var initializers = []setupFunc{setupConfig, setupLogger}
var healthchecks = map[string]healthFunc{}
var starters = []func(ctx context.Context, app *Application) error{}

type Application struct {
	Config *Config
	Log    *zap.SugaredLogger

	//golem:template:app/app_partial_definitions
	// DO NOT EDIT. This section is managed by github.com/dashotv/golem.
	// Models
	DB *Connector

	//golem:template:app/app_partial_definitions

	Bot *Bot
}

func Setup() error {
	if app != nil {
		return errors.New("application already setup")
	}

	app = &Application{}

	for _, f := range initializers {
		if err := f(app); err != nil {
			return err
		}
	}

	return nil
}

func Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if app == nil {
		if err := Setup(); err != nil {
			return err
		}
	}

	for _, f := range starters {
		if err := f(ctx, app); err != nil {
			return err
		}
	}

	app.Log.Info("started")

	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	for {
		select {
		case <-sc:
		case <-ctx.Done():
			app.Log.Info("stopping")
			return nil
		}
	}
}

func (a *Application) Health() (map[string]bool, error) {
	resp := make(map[string]bool)
	for n, f := range healthchecks {
		err := f(a)
		resp[n] = err == nil
	}

	return resp, nil
}
