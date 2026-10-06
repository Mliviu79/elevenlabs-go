package elevenlabs

import (
	"testing"
	"time"
)

type requestClientCase struct {
	name   string
	client *Client
	want   time.Duration
}

// requestClientCases returns one case for each way a Client is given its request timeout and no
// supplied client. The default client's previous timeout is restored when the test ends.
func requestClientCases(tb testing.TB) []requestClientCase {
	tb.Helper()
	const setTimeout = 45 * time.Second
	previous := getDefaultClient().timeout
	tb.Cleanup(func() { SetTimeout(previous) })
	SetTimeout(setTimeout)
	return []requestClientCase{
		{name: "constructed with a request timeout", client: &Client{timeout: 30 * time.Second}, want: 30 * time.Second},
		{name: "default client after SetTimeout", client: getDefaultClient(), want: setTimeout},
	}
}

// TestRequestClient verifies that a request goes through a client whose Timeout is the request
// timeout the Client was given, including one SetTimeout changed after construction.
func TestRequestClient(t *testing.T) {
	for _, tc := range requestClientCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.client.requestClient().Timeout; got != tc.want {
				t.Errorf("requestClient().Timeout = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRequestClientNeverUnbounded verifies the invariant that, with no supplied client, no request
// goes through an http.Client without a Timeout.
func TestRequestClientNeverUnbounded(t *testing.T) {
	for _, tc := range requestClientCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.client.requestClient().Timeout; got == 0 {
				t.Errorf("requestClient().Timeout = %v, want a non-zero bound", got)
			}
		})
	}
}
