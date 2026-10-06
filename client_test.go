package elevenlabs

import (
	"net/http"
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

// TestRequestClientWithHTTPClient verifies that a supplied client is the one every request goes
// through, unmodified, and that a nil one keeps the client bounded by the request timeout.
func TestRequestClientWithHTTPClient(t *testing.T) {
	supplied := &http.Client{}
	testCases := []struct {
		name        string
		hc          *http.Client
		wantClient  *http.Client
		wantTimeout time.Duration
	}{
		{name: "supplied client", hc: supplied, wantClient: supplied},
		{name: "nil supplied client", hc: nil, wantTimeout: 30 * time.Second},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{timeout: 30 * time.Second}
			WithHTTPClient(tc.hc)(c)
			got := c.requestClient()
			if tc.wantClient != nil {
				if got != tc.wantClient {
					t.Errorf("requestClient() = %p, want the supplied client %p", got, tc.wantClient)
				}
				return
			}
			if got == nil || got.Timeout != tc.wantTimeout {
				t.Errorf("requestClient() = %+v, want a client with Timeout %v", got, tc.wantTimeout)
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
