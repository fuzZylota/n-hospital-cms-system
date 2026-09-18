package main

import (
	"context"
	"errors"
	"fmt"
	"lib/notificationhub"
	"models/notify"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
)

func TestHubLifecycleOrderAndAllFailureCombinations(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			var events []string
			var wantErrors []string
			raw := errors.New("private transport database secret")
			if mask&1 != 0 {
				wantErrors = append(wantErrors, "HTTP listen failed")
			}
			if mask&2 != 0 {
				wantErrors = append(wantErrors, "HTTP shutdown failed")
			}
			if mask&4 != 0 {
				wantErrors = append(wantErrors, "notification hub shutdown failed")
			}
			err := runLifecycle(context.Background(), bootstrap{
				openLegacy: func() (func(), error) {
					events = append(events, "legacy")
					return func() { events = append(events, "close legacy") }, nil
				},
				openOwned: func(context.Context) (func(), error) {
					events = append(events, "owned")
					return func() { events = append(events, "close owned") }, nil
				},
				openHub: func() (func(context.Context) error, error) {
					events = append(events, "hub")
					return func(ctx context.Context) error {
						events = append(events, "close hub")
						if _, ok := ctx.Deadline(); !ok || ctx.Err() != nil {
							t.Error("hub lacks fresh shutdown budget")
						}
						if mask&4 != 0 {
							return raw
						}
						return nil
					}, nil
				},
				newServer: func() (httpLifecycle, error) {
					events = append(events, "HTTP")
					return httpLifecycle{listen: func() error {
						if mask&1 != 0 {
							return raw
						}
						return nil
					}, shutdown: func(context.Context) error {
						events = append(events, "close HTTP")
						if mask&2 != 0 {
							return raw
						}
						return nil
					}}, nil
				},
			})
			if !reflect.DeepEqual(events, []string{"legacy", "owned", "hub", "HTTP", "close HTTP", "close hub", "close owned", "close legacy"}) {
				t.Fatal(events)
			}
			if len(wantErrors) == 0 {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != strings.Join(wantErrors, "\n") || errors.Is(err, raw) {
				t.Fatalf("unsafe/wrong primary stage: %v", err)
			}
		})
	}
}

func TestHubStartupAndHTTPStartupFailureCleanup(t *testing.T) {
	for _, invalid := range []bool{true, false} {
		t.Run(fmt.Sprint(invalid), func(t *testing.T) {
			var events []string
			err := runLifecycle(context.Background(), bootstrap{
				openLegacy: func() (func(), error) { return func() { events = append(events, "legacy") }, nil },
				openOwned:  func(context.Context) (func(), error) { return func() { events = append(events, "owned") }, nil },
				openHub: func() (func(context.Context) error, error) {
					if invalid {
						_, err := notificationhub.New(0)
						return nil, err
					}
					h, _ := notificationhub.New(1)
					return func(ctx context.Context) error { events = append(events, "hub"); return h.Shutdown(ctx) }, nil
				},
				newServer: func() (httpLifecycle, error) {
					if invalid {
						t.Fatal("HTTP started after invalid capacity")
					}
					return httpLifecycle{}, errors.New("private bind")
				},
			})
			want := []string{"hub", "owned", "legacy"}
			stage := "HTTP startup failed"
			if invalid {
				want = []string{"owned", "legacy"}
				stage = "notification hub startup failed"
			}
			if !reflect.DeepEqual(events, want) || err == nil || err.Error() != stage {
				t.Fatal("startup cleanup/stage")
			}
		})
	}
}

func TestHubShutdownFreshAfterExpiredHTTPAndBrokenSender(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := notificationhub.New(1)
		started := make(chan struct{})
		release := make(chan struct{})
		r, _ := h.Register(context.Background(), "test", notify.Client{ID: "id"}, func(context.Context, []byte) error { close(started); <-release; return nil })
		h.Broadcast("test", []byte("private"), nil)
		<-started
		var events []string
		err := runLifecycle(context.Background(), bootstrap{
			openLegacy: func() (func(), error) { return func() { events = append(events, "legacy") }, nil },
			openOwned:  func(context.Context) (func(), error) { return func() { events = append(events, "owned") }, nil },
			openHub: func() (func(context.Context) error, error) {
				return func(ctx context.Context) error {
					events = append(events, "hub")
					if ctx.Err() != nil {
						t.Error("HTTP context reused")
					}
					return h.Shutdown(ctx)
				}, nil
			},
			newServer: func() (httpLifecycle, error) {
				return httpLifecycle{listen: func() error { return nil }, shutdown: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}, nil
			},
		})
		close(release)
		<-r.Done()
		if !reflect.DeepEqual(events, []string{"hub", "owned", "legacy"}) || err == nil || err.Error() != "HTTP shutdown failed\nnotification hub shutdown failed" {
			t.Fatal("timeout hid failure or skipped cleanup")
		}
	})
}
