package notificationws

// These tests require the real, unchanged Fiber/WebSocket dependencies. They
// must remain BLOCKED when the approved offline cache lacks transitive inputs;
// the fake Socket tests are not a replacement for this transport evidence.
import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"models/notify"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	fastws "github.com/fasthttp/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type signaledConn struct {
	net.Conn
	writing chan struct{}
	once    sync.Once
}

func (c *signaledConn) Write(p []byte) (int, error) {
	c.once.Do(func() { close(c.writing) })
	return c.Conn.Write(p)
}

// A real WebSocket client over net.Pipe; the peer completes the HTTP handshake
// and then deliberately stops reading. No external network or application runs.
func pipeWebSocket(t *testing.T) (*fastws.Conn, net.Conn, *signaledConn) {
	t.Helper()
	local, peer := net.Pipe()
	signal := &signaledConn{Conn: local, writing: make(chan struct{})}
	peerDone := make(chan struct{})
	go func() {
		defer close(peerDone)
		request, err := http.ReadRequest(bufio.NewReader(peer))
		if err != nil {
			return
		}
		sum := sha1.Sum([]byte(request.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		fmt.Fprintf(peer, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", base64.StdEncoding.EncodeToString(sum[:]))
	}()
	dialer := fastws.Dialer{NetDialContext: func(context.Context, string, string) (net.Conn, error) { return signal, nil }, HandshakeTimeout: time.Second}
	c, _, err := dialer.Dial("ws://pipe.test/", nil)
	if err != nil {
		peer.Close()
		local.Close()
		t.Fatal("pipe handshake failed")
	}
	wait(t, peerDone)
	// Reset the signal after handshake; the transport has not started yet.
	signal.once = sync.Once{}
	signal.writing = make(chan struct{})
	t.Cleanup(func() { c.Close(); peer.Close() })
	return c, peer, signal
}

func TestRealWebSocketPipeCancellation(t *testing.T) {
	for _, mode := range []string{"context", "hub"} {
		t.Run(mode, func(t *testing.T) {
			c, _, signal := pipeWebSocket(t)
			tr := newTransport(c)
			h := hub(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := h.Register(ctx, Notifications, notify.Client{ID: "pipe"}, tr.send)
			if err != nil {
				t.Fatal(err)
			}
			go tr.watch(r.Done())
			readDone := make(chan struct{})
			go func() { defer close(readDone); _, _, _ = c.ReadMessage() }()
			h.Broadcast(Notifications, []byte(`{"message":"pipe"}`), nil)
			wait(t, signal.writing)
			if mode == "context" {
				cancel()
			} else {
				end, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				if h.Shutdown(end) != nil {
					t.Fatal("real blocked send did not cancel")
				}
			}
			wait(t, r.Done())
			wait(t, tr.watchDone)
			wait(t, readDone)
		})
	}
}

func TestRealFiberAuthLocalsAndUpgradeRefusal(t *testing.T) {
	for _, authorized := range []bool{true, false} {
		t.Run(fmt.Sprint(authorized), func(t *testing.T) {
			h := observe(hub(t))
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Get("/notifications", Handler(h, func(c *fiber.Ctx) (notify.UserID, error) {
				if h.calls.Load() != 0 {
					t.Error("registration before authentication")
				}
				if !authorized {
					return "", errors.New("private auth")
				}
				return notify.UserID("42"), nil
			}, nil, websocket.Config{Subprotocols: []string{"kullanici"}}))
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal("test listener failed")
			}
			serverDone := make(chan struct{})
			go func() { defer close(serverDone); _ = app.Listener(listener) }()
			defer func() {
				listener.Close()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = app.ShutdownWithContext(ctx)
				wait(t, serverDone)
			}()
			dialer := fastws.Dialer{Subprotocols: []string{"kullanici"}, HandshakeTimeout: time.Second}
			c, response, err := dialer.Dial("ws://"+listener.Addr().String()+"/notifications", nil)
			if !authorized {
				if c != nil {
					c.Close()
				}
				if err == nil || response == nil || response.StatusCode != 401 || h.calls.Load() != 0 {
					t.Fatal("unauthenticated upgrade registered")
				}
				response.Body.Close()
				return
			}
			if err != nil {
				t.Fatal("authorized upgrade failed")
			}
			defer c.Close()
			var client notify.Client
			select {
			case client = <-h.registered:
			case <-time.After(time.Second):
				t.Fatal("no registration")
			}
			r := <-h.registration
			if client.Metadata.UserID != "42" || client.Metadata.Protocol != recipientProtocol {
				t.Fatal("locals identity not transferred")
			}
			h.Broadcast(Notifications, []byte(`{"message":"once"}`), nil)
			c.SetReadDeadline(time.Now().Add(time.Second))
			kind, payload, err := c.ReadMessage()
			if err != nil || kind != fastws.TextMessage || string(payload) != `{"message":"once"}` {
				t.Fatal("text payload double encoded")
			}
			if c.WriteMessage(fastws.TextMessage, []byte(`{"uid":"attacker"}`)) != nil {
				t.Fatal("write failed")
			}
			wait(t, r.Done())
		})
	}
}

func TestRealFiberFailedHandshakeDoesNotRegister(t *testing.T) {
	h := observe(hub(t))
	app := fiber.New()
	app.Get("/notifications", Handler(h, func(*fiber.Ctx) (notify.UserID, error) { return "42", nil }, nil, websocket.Config{}))
	request, _ := http.NewRequest(http.MethodGet, "http://test/notifications", strings.NewReader(""))
	response, err := app.Test(request)
	if err != nil {
		t.Fatal("request failed")
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
	if h.calls.Load() != 0 || response.StatusCode != 426 {
		t.Fatal("failed upgrade registered")
	}
}

// No listener is needed for rejection before upgrade. Still requires real Fiber
// dependencies and is kept BLOCKED alongside the other integration tests.
func TestRealFiberInvalidUIDBeforeUpgrade(t *testing.T) {
	for _, tc := range uidCases() {
		uid, typed := tc.value.(notify.UserID)
		if !typed || tc.want != "" {
			continue // Handler's public auth callback cannot return another type.
		}
		t.Run(tc.name, func(t *testing.T) {
			h := observe(hub(t))
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			authCalls := 0
			app.Use(func(c *fiber.Ctx) error {
				err := c.Next()
				if c.Locals(identityKey) != nil || c.Locals("notification.event") != nil {
					t.Error("invalid UID wrote upgrade locals")
				}
				return err
			})
			app.Get("/notifications", Handler(h, func(*fiber.Ctx) (notify.UserID, error) {
				authCalls++
				return uid, nil
			}, nil, websocket.Config{}))
			request, _ := http.NewRequest(http.MethodGet, "http://test/notifications", nil)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal("in-memory request failed")
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 401 || string(body) != fiber.ErrUnauthorized.Message || authCalls != 1 || h.calls.Load() != 0 {
				t.Fatal("invalid UID refusal was unsafe or reached registration")
			}
		})
	}
}
