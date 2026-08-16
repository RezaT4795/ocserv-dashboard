package occtl

import (
	"errors"
	"testing"
)

func TestNormalizeDisconnectUserResultTreatsNoActiveSessionAsSuccess(t *testing.T) {
	output := "could not disconnect user 'expired-user'\n"

	gotOutput, err := normalizeDisconnectUserResult(
		output,
		errors.New("exit status 1"),
	)

	if err != nil {
		t.Fatalf(
			"normalizeDisconnectUserResult() error = %v, want nil",
			err,
		)
	}

	if gotOutput != output {
		t.Fatalf(
			"normalizeDisconnectUserResult() output = %q, want %q",
			gotOutput,
			output,
		)
	}
}

func TestNormalizeDisconnectUserResultPreservesRealFailure(t *testing.T) {
	commandErr := errors.New("exit status 1")
	output := "error connecting to ocserv socket\n"

	gotOutput, err := normalizeDisconnectUserResult(
		output,
		commandErr,
	)

	if !errors.Is(err, commandErr) {
		t.Fatalf(
			"normalizeDisconnectUserResult() error = %v, want %v",
			err,
			commandErr,
		)
	}

	if gotOutput != output {
		t.Fatalf(
			"normalizeDisconnectUserResult() output = %q, want %q",
			gotOutput,
			output,
		)
	}
}

func TestNormalizeDisconnectUserResultPreservesSuccessfulCommand(t *testing.T) {
	output := "user disconnected\n"

	gotOutput, err := normalizeDisconnectUserResult(output, nil)

	if err != nil {
		t.Fatalf(
			"normalizeDisconnectUserResult() error = %v, want nil",
			err,
		)
	}

	if gotOutput != output {
		t.Fatalf(
			"normalizeDisconnectUserResult() output = %q, want %q",
			gotOutput,
			output,
		)
	}
}
