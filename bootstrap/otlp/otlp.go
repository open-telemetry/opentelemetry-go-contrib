package otlp

import (
	"context"
	"errors"
	"log"
)

type ShutdownFunc func(ctx context.Context) error
type ShutdownIgnoreErrorFunc func(ctx context.Context)

func emptyShutdown(ctx context.Context) error {
	return nil
}

func wrapShutdown(shutdownFuncs []ShutdownFunc) ShutdownIgnoreErrorFunc {
	return func(ctx context.Context) {

		var errs []error

		for _, fn := range shutdownFuncs {
			err := fn(ctx)

			if err != nil {
				errs = append(errs, err)
			}
		}

		// ignore error
		if len(errs) > 0 {
			log.Println("failed to shutdown:", errors.Join(errs...))
		}
	}
}

func Setup(ctx context.Context) ShutdownIgnoreErrorFunc {
	var shutdownFuncs []ShutdownFunc

	sf, err := setupTrace(ctx)

	// ignore error
	if err != nil {
		log.Println("failed to init trace:", err)
	}

	shutdownFuncs = append(shutdownFuncs, sf)

	sf, err = setupLog(ctx)

	// ignore error
	if err != nil {
		log.Println("failed to init log:", err)
	}

	shutdownFuncs = append(shutdownFuncs, sf)

	return wrapShutdown(shutdownFuncs)
}
