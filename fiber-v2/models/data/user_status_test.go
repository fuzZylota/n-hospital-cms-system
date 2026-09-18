package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"models/data"
)

type userStatusReaderFunc func(context.Context, string) (data.UserStatus, error)

func (f userStatusReaderFunc) LookupUserStatus(ctx context.Context, userID string) (data.UserStatus, error) {
	return f(ctx, userID)
}

var _ data.UserStatusReader = userStatusReaderFunc(nil)

func TestUserStatusContract(t *testing.T) {
	assertStructFields(t, data.UserStatus{}, []fieldContract{
		{name: "Found", typeOf: reflect.TypeOf(false)},
		{name: "Active", typeOf: reflect.TypeOf(false)},
		{name: "Role", typeOf: reflect.TypeOf("")},
	})
}

func TestUserStatusReaderOutcomes(t *testing.T) {
	repositoryFailure := errors.New("safe user lookup failure")

	tests := []struct {
		name      string
		context   func() context.Context
		reader    data.UserStatusReader
		want      data.UserStatus
		wantError error
	}{
		{
			name:    "active user",
			context: context.Background,
			reader: userStatusReaderFunc(func(_ context.Context, userID string) (data.UserStatus, error) {
				if userID != "42" {
					t.Fatalf("userID = %q, want 42", userID)
				}
				return data.UserStatus{Found: true, Active: true, Role: "admin"}, nil
			}),
			want: data.UserStatus{Found: true, Active: true, Role: "admin"},
		},
		{
			name:    "not found",
			context: context.Background,
			reader: userStatusReaderFunc(func(context.Context, string) (data.UserStatus, error) {
				return data.UserStatus{Found: false}, nil
			}),
			want: data.UserStatus{Found: false},
		},
		{
			name:    "inactive user retains current role",
			context: context.Background,
			reader: userStatusReaderFunc(func(context.Context, string) (data.UserStatus, error) {
				return data.UserStatus{Found: true, Active: false, Role: "moderator"}, nil
			}),
			want: data.UserStatus{Found: true, Active: false, Role: "moderator"},
		},
		{
			name:    "lookup error is neither success nor not found",
			context: context.Background,
			reader: userStatusReaderFunc(func(context.Context, string) (data.UserStatus, error) {
				return data.UserStatus{}, repositoryFailure
			}),
			wantError: repositoryFailure,
		},
		{
			name: "context canceled",
			context: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			reader:    userStatusReaderFunc(contextAwareUserStatus),
			wantError: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			context: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			reader:    userStatusReaderFunc(contextAwareUserStatus),
			wantError: context.DeadlineExceeded,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.reader.LookupUserStatus(test.context(), "42")
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("result = %#v, want %#v", got, test.want)
			}
			if err != nil && got.Found {
				t.Fatal("lookup error was represented as a found user")
			}
		})
	}
}

func contextAwareUserStatus(ctx context.Context, _ string) (data.UserStatus, error) {
	if err := ctx.Err(); err != nil {
		return data.UserStatus{}, err
	}
	return data.UserStatus{Found: false}, nil
}
