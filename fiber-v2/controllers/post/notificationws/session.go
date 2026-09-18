package notificationws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"models/notify"
)

const identityKey = "notification.authenticated.uid"
const Notifications notify.RoomID = "notifications"
const producers notify.RoomID = "notification-events"
const recipientProtocol notify.Protocol = "kullanici"

type Event uint8

const (
	Subscriber Event = iota
	Appointment
	Application
	Contact
	Unknown
)

// The header remains an untrusted event selector (N08). It is never a user,
// role or branch credential, and is not copied into recipient metadata.
func eventFor(protocol string) Event {
	switch protocol {
	case "kullanici":
		return Subscriber
	case "randevu":
		return Appointment
	case "is-basvurusu":
		return Application
	case "iletisim":
		return Contact
	default:
		return Unknown
	}
}

func identity(value any) (notify.Metadata, error) {
	uid, err := authenticatedUserID(value)
	if err != nil {
		return notify.Metadata{}, err
	}
	return notify.Metadata{UserID: uid, Protocol: recipientProtocol}, nil
}

func connectionID(random io.Reader) (notify.ConnectionID, error) {
	var bytes [32]byte
	if _, err := io.ReadFull(random, bytes[:]); err != nil {
		return "", errRandom
	}
	return notify.ConnectionID(hex.EncodeToString(bytes[:])), nil
}

// serve owns read/close and waits for the hub writer and watcher before the
// Fiber wrapper is returned to its pool. Even panic/Goexit runs this cleanup.
func serve(hub notify.Hub, socket Socket, local any, event Event, onText func(Event, []byte), random io.Reader) (err error) {
	t := newTransport(socket)
	defer t.close()
	defer func() {
		if recover() != nil {
			err = errHandler
		}
	}()
	metadata, err := identity(local)
	if err != nil {
		return err
	}
	id, err := connectionID(random)
	if err != nil {
		return err
	}
	room := Notifications
	if event != Subscriber {
		room = producers
	}
	if hub == nil {
		return errRegistration
	}
	registration, err := hub.Register(context.Background(), room, notify.Client{ID: id, Metadata: metadata}, t.send)
	if err != nil {
		return errRegistration
	}
	go t.watch(registration.Done())
	defer func() {
		registration.Unregister()
		t.close() // Interrupt read/write before waiting, including panic cleanup.
		<-registration.Done()
		<-t.watchDone
	}()
	for {
		kind, payload, readErr := socket.ReadMessage()
		if readErr != nil {
			return nil
		}
		if kind == textMessage {
			if onText != nil {
				onText(event, payload)
			}
			return nil // Preserve the legacy one inbound text message lifecycle.
		}
	}
}

func serveAuthenticated(hub notify.Hub, socket Socket, local any, event Event, onText func(Event, []byte)) error {
	return serve(hub, socket, local, event, onText, rand.Reader)
}
