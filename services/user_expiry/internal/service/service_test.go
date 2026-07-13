package service

import (
	"errors"
	"strings"
	"testing"
)

func TestEnforceExpiredUserLocksBeforeDisconnecting(
	t *testing.T,
) {
	var calls []string

	service := &CornService{
		lockUser: func(
			username string,
		) (string, error) {
			calls = append(
				calls,
				"lock:"+username,
			)

			return "", nil
		},
		disconnectUser: func(
			username string,
		) (string, error) {
			calls = append(
				calls,
				"disconnect:"+username,
			)

			return "", nil
		},
	}

	if err := service.enforceExpiredUser(
		"expired-user",
	); err != nil {
		t.Fatalf(
			"enforceExpiredUser() error = %v",
			err,
		)
	}

	want := []string{
		"lock:expired-user",
		"disconnect:expired-user",
	}

	if len(calls) != len(want) {
		t.Fatalf(
			"calls = %v, want %v",
			calls,
			want,
		)
	}

	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf(
				"calls = %v, want %v",
				calls,
				want,
			)
		}
	}
}

func TestEnforceExpiredUserAttemptsBothActionsWhenLockFails(
	t *testing.T,
) {
	disconnectCalled := false

	service := &CornService{
		lockUser: func(
			string,
		) (string, error) {
			return "", errors.New(
				"lock failed",
			)
		},
		disconnectUser: func(
			string,
		) (string, error) {
			disconnectCalled = true

			return "", nil
		},
	}

	err := service.enforceExpiredUser(
		"expired-user",
	)

	if err == nil {
		t.Fatal(
			"enforceExpiredUser() error = nil, want error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"lock authentication: lock failed",
	) {
		t.Fatalf(
			"enforceExpiredUser() error = %q",
			err,
		)
	}

	if !disconnectCalled {
		t.Fatal(
			"disconnect was not attempted after lock failure",
		)
	}
}

func TestEnforceExpiredUserReturnsBothFailures(
	t *testing.T,
) {
	service := &CornService{
		lockUser: func(
			string,
		) (string, error) {
			return "", errors.New(
				"lock failed",
			)
		},
		disconnectUser: func(
			string,
		) (string, error) {
			return "", errors.New(
				"disconnect failed",
			)
		},
	}

	err := service.enforceExpiredUser(
		"expired-user",
	)

	if err == nil {
		t.Fatal(
			"enforceExpiredUser() error = nil, want error",
		)
	}

	for _, message := range []string{
		"lock authentication: lock failed",
		"disconnect sessions: disconnect failed",
	} {
		if !strings.Contains(
			err.Error(),
			message,
		) {
			t.Fatalf(
				"enforceExpiredUser() error = %q, want %q",
				err,
				message,
			)
		}
	}
}
