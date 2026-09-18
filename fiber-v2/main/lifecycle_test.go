package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestLifecycleOwnershipAndOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		want []string
	}{
		{"legacy error", []string{"legacy"}},
		{"owned error", []string{"legacy", "owned", "close legacy"}},
		{"bind error", []string{"legacy", "owned", "server", "close owned", "close legacy"}},
		{"listen error", []string{"legacy", "owned", "server", "shutdown", "close owned", "close legacy"}},
		{"signal", []string{"legacy", "owned", "server", "shutdown", "close owned", "close legacy"}},
		{"shutdown error", []string{"legacy", "owned", "server", "shutdown", "close owned", "close legacy"}},
		{"signal before listen", []string{"legacy", "owned", "server", "shutdown", "close owned", "close legacy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			backend := errors.New("private-backend password")
			stopped := make(chan struct{})
			finished := make(chan struct{})
			listened := false
			err := runLifecycle(ctx, bootstrap{
				openLegacy: func() (func(), error) {
					events = append(events, "legacy")
					if tc.name == "legacy error" {
						return nil, backend
					}
					return func() { events = append(events, "close legacy") }, nil
				},
				openOwned: func(context.Context) (func(), error) {
					events = append(events, "owned")
					if tc.name == "owned error" {
						return nil, errors.New("database pool startup failed: ping")
					}
					return func() { events = append(events, "close owned") }, nil
				},
				newServer: func() (httpLifecycle, error) {
					events = append(events, "server")
					if tc.name == "bind error" {
						return httpLifecycle{}, backend
					}
					if tc.name == "signal before listen" {
						cancel()
					}
					return httpLifecycle{
						listen: func() error {
							defer close(finished)
							if tc.name == "listen error" {
								return backend
							}
							cancel()
							<-stopped
							return nil
						},
						shutdown: func(ctx context.Context) error {
							events = append(events, "shutdown")
							deadline, ok := ctx.Deadline()
							if !ok || time.Until(deadline) > shutdownTimeout || ctx.Err() != nil {
								t.Error("shutdown lacks fresh bounded context")
							}
							close(stopped)
							if tc.name != "signal before listen" {
								<-finished
								listened = true
							}
							if tc.name == "shutdown error" {
								return backend
							}
							return nil
						},
					}, nil
				},
			})
			if !reflect.DeepEqual(events, tc.want) {
				t.Fatalf("events=%v want=%v", events, tc.want)
			}
			wantErr := tc.name != "signal" && tc.name != "signal before listen"
			if (err != nil) != wantErr {
				t.Fatalf("error=%v", err)
			}
			if err != nil && strings.Contains(err.Error(), "password") {
				t.Fatal("backend error leaked")
			}
			if tc.name == "signal before listen" && listened {
				t.Fatal("listener started after cancellation")
			}
		})
	}
}

type memoryListener struct{ closes atomic.Int32 }

func (*memoryListener) Accept() (net.Conn, error) { return nil, errors.New("unused") }
func (l *memoryListener) Close() error            { l.closes.Add(1); return nil }
func (*memoryListener) Addr() net.Addr            { return nil }

func TestListenerCloseIsOnceAcrossOwners(t *testing.T) {
	raw := &memoryListener{}
	listener := &onceListener{Listener: raw}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); listener.Close() }()
	}
	wg.Wait()
	if raw.closes.Load() != 1 {
		t.Fatal("socket closed more than once")
	}
}

func TestShutdownDeadlineStillClosesPools(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		start := time.Now()
		closed := 0
		err := runLifecycle(ctx, bootstrap{
			openLegacy: func() (func(), error) { return func() { closed++ }, nil },
			openOwned:  func(context.Context) (func(), error) { return func() { closed++ }, nil },
			newServer: func() (httpLifecycle, error) {
				return httpLifecycle{
					listen:   func() error { cancel(); <-ctx.Done(); return nil },
					shutdown: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
				}, nil
			},
		})
		if err == nil || closed != 2 || time.Since(start) != shutdownTimeout {
			t.Fatalf("shutdown result err=%v closes=%d elapsed=%s", err, closed, time.Since(start))
		}
	})
}

func TestLifecyclePrimaryAndSecondaryFailures(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		listenFailure, shutdownFailure bool
		want                           string
	}{
		{"listen", true, false, "HTTP listen failed"},
		{"both", true, true, "HTTP listen failed\nHTTP shutdown failed"},
		{"shutdown", false, true, "HTTP shutdown failed"},
		{"signal", false, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			finished := make(chan struct{})
			backend := errors.New("private-backend password")
			err := runLifecycle(ctx, bootstrap{
				openLegacy: func() (func(), error) { return func() { events = append(events, "legacy") }, nil },
				openOwned:  func(context.Context) (func(), error) { return func() { events = append(events, "owned") }, nil },
				newServer: func() (httpLifecycle, error) {
					return httpLifecycle{
						listen: func() error {
							defer close(finished)
							if tc.listenFailure {
								return backend
							}
							cancel()
							return nil
						},
						shutdown: func(context.Context) error {
							<-finished
							events = append(events, "HTTP")
							if tc.shutdownFailure {
								return backend
							}
							return nil
						},
					}, nil
				},
			})
			if !reflect.DeepEqual(events, []string{"HTTP", "owned", "legacy"}) {
				t.Fatal("cleanup order/count changed")
			}
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || err.Error() != tc.want {
				t.Fatalf("wrong primary/secondary stage: %v", err)
			}
			var check func(error)
			check = func(err error) {
				if errors.Is(err, backend) {
					t.Fatal("raw backend retained")
				}
				for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
					if strings.Contains(fmt.Sprintf(format, err), "password") {
						t.Fatal("raw backend leaked")
					}
				}
				if joined, ok := err.(interface{ Unwrap() []error }); ok {
					for _, child := range joined.Unwrap() {
						check(child)
					}
				} else if child := errors.Unwrap(err); child != nil {
					check(child)
				}
			}
			check(err)
		})
	}
}
