package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

const shutdownTimeout = 10 * time.Second

// bootstrap is the narrow composition seam. Openers transfer ownership only
// on success. A failed owned opener closes its own partially acquired pool.
type bootstrap struct {
	openLegacy func() (close func(), err error)
	openOwned  func(context.Context) (close func(), err error)
	newServer  func() (httpLifecycle, error)
}

// Fiber and the composition root may both request Close; the socket is closed once.
type onceListener struct {
	net.Listener
	once sync.Once
	err  error
}

func (l *onceListener) Close() error {
	l.once.Do(func() { l.err = l.Listener.Close() })
	return l.err
}

type httpLifecycle struct {
	listen   func() error
	shutdown func(context.Context) error
}

func runLifecycle(ctx context.Context, boot bootstrap) (err error) {
	closeLegacy, err := boot.openLegacy()
	if err != nil {
		return errors.New("legacy database startup failed")
	}
	defer closeLegacy()
	closeOwned, err := boot.openOwned(ctx)
	if err != nil {
		return err // Owned opener already supplies a safe staged error.
	}
	defer closeOwned()
	server, err := boot.newServer()
	if err != nil {
		return errors.New("HTTP startup failed")
	}
	// Shutdown precedes both pool closes on every path, including Listen errors.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if shutdownErr := server.shutdown(shutdownCtx); shutdownErr != nil {
			if err == nil {
				err = errors.New("HTTP shutdown failed")
			} else {
				// The primary is already a safe stage; never retain raw errors.
				err = errors.Join(err, errors.New("HTTP shutdown failed"))
			}
		}
	}()
	if ctx.Err() != nil {
		return nil
	}
	listened := make(chan error, 1)
	go func() { listened <- server.listen() }()
	select {
	case listenErr := <-listened:
		if listenErr != nil {
			return errors.New("HTTP listen failed")
		}
		return nil
	case <-ctx.Done():
		return nil
	}
}
