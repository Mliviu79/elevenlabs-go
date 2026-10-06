//go:build go1.25

// testing/synctest refuses to run under the timer channels the module's go 1.18 directive selects
// (those from before Go 1.23); this selects the current ones for this package's test binary only.
//go:debug asynctimerchan=0

package elevenlabs

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"testing/synctest"
	"time"
)

// unservedBaseURL has a scheme http.DefaultTransport refuses at once, so a request that bypassed
// the supplied transport fails without opening a socket inside the bubble.
const unservedBaseURL = "elevenlabs-test://unserved"

// deadlineTransport holds each request until its context ends, recording when the request arrived
// and the deadline its context carried; it returns the context's error unwrapped.
type deadlineTransport struct {
	arrivedAt   time.Time
	deadline    time.Time
	hasDeadline bool
	ended       chan struct{}
}

func (tr *deadlineTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close() // the request body is an in-memory buffer; closing it cannot fail
	}
	ctx := req.Context()
	tr.arrivedAt = time.Now()
	tr.deadline, tr.hasDeadline = ctx.Deadline()
	<-ctx.Done()
	close(tr.ended)
	return nil, ctx.Err()
}

// TestTextToSpeechStreamSuppliedClientTimeout verifies that a supplied client's Timeout, shorter
// than the request timeout, ends a held response as a timeout at the supplied client's deadline.
func TestTextToSpeechStreamSuppliedClientTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const suppliedTimeout, requestTimeout = 2 * time.Second, 30 * time.Second
		tr := &deadlineTransport{ended: make(chan struct{})}
		hc := &http.Client{Transport: tr, Timeout: suppliedTimeout}
		client := NewMockClient(t.Context(), unservedBaseURL, testAPIKey, requestTimeout, WithHTTPClient(hc))

		err := client.TextToSpeechStream(&signalWriter{firstWrite: make(chan struct{})}, testVoiceID, TextToSpeechRequest{Text: "hello"})

		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Errorf("TextToSpeechStream() = %v, want a net.Error whose Timeout() is true", err)
		}
		select {
		case <-tr.ended:
		default:
			t.Fatal("the supplied transport never saw its request context end")
		}
		if want := tr.arrivedAt.Add(suppliedTimeout); !tr.hasDeadline || !tr.deadline.Equal(want) {
			t.Errorf("request context deadline = %v (set %t), want the supplied client's deadline %v", tr.deadline, tr.hasDeadline, want)
		}
	})
}
