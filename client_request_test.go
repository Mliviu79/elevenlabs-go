//go:build go1.24

package elevenlabs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAPIKey         = "fake-test-key"
	testVoiceID        = "voice-under-test"
	testRequestTimeout = 30 * time.Second
	testWaitBound      = 5 * time.Second
)

// countingServer starts a server that answers every request with body and counts the requests it
// served; it is closed when the test ends.
func countingServer(tb testing.TB, body string) (*httptest.Server, *int32) {
	tb.Helper()
	served := new(int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(served, 1)
		_, _ = io.WriteString(w, body) // best-effort: a failed write shows up as the client's wrong body
	}))
	tb.Cleanup(srv.Close)
	return srv, served
}

// recordingTransport answers every request with a 200 and body, recording each request it served.
type recordingTransport struct {
	body string

	mu       sync.Mutex
	requests []*http.Request
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close() // the request body is an in-memory buffer; closing it cannot fail
	}
	rt.mu.Lock()
	rt.requests = append(rt.requests, req)
	rt.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(rt.body)),
		Request:    req,
	}, nil
}

func (rt *recordingTransport) served() []*http.Request {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]*http.Request(nil), rt.requests...)
}

// TestTextToSpeechStreamWithHTTPClient verifies that a supplied client serves every request in place
// of the default one, and that a nil supplied client leaves the default in place.
func TestTextToSpeechStreamWithHTTPClient(t *testing.T) {
	const transportBody, serverBody = "audio from the supplied transport", "audio from the server"
	testCases := []struct {
		name           string
		supply         bool
		wantBody       string
		wantServed     int32
		wantRoundTrips int
	}{
		{name: "supplied client", supply: true, wantBody: transportBody, wantServed: 0, wantRoundTrips: 1},
		{name: "nil supplied client", supply: false, wantBody: serverBody, wantServed: 1, wantRoundTrips: 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, served := countingServer(t, serverBody)
			rec := &recordingTransport{body: transportBody}
			var hc *http.Client
			if tc.supply {
				hc = &http.Client{Transport: rec}
			}
			client := NewMockClient(t.Context(), srv.URL, testAPIKey, testRequestTimeout, WithHTTPClient(hc))

			var w bytes.Buffer
			if err := client.TextToSpeechStream(&w, testVoiceID, TextToSpeechRequest{Text: "hello"}); err != nil {
				t.Fatalf("TextToSpeechStream() = %v, want nil", err)
			}

			if got := w.String(); got != tc.wantBody {
				t.Errorf("TextToSpeechStream() wrote %q, want %q", got, tc.wantBody)
			}
			if got := atomic.LoadInt32(served); got != tc.wantServed {
				t.Errorf("server served %d requests, want %d", got, tc.wantServed)
			}
			requests := rec.served()
			if len(requests) != tc.wantRoundTrips {
				t.Fatalf("supplied transport served %d requests, want %d", len(requests), tc.wantRoundTrips)
			}
			for _, req := range requests {
				if req.Method != http.MethodPost {
					t.Errorf("request method = %q, want %q", req.Method, http.MethodPost)
				}
				if wantSuffix := "/text-to-speech/" + testVoiceID + "/stream"; !strings.HasSuffix(req.URL.Path, wantSuffix) {
					t.Errorf("request path = %q, want a path ending %q", req.URL.Path, wantSuffix)
				}
				if got := req.Header.Get("xi-api-key"); got != testAPIKey {
					t.Errorf("request xi-api-key = %q, want %q", got, testAPIKey)
				}
			}
		})
	}
}

// holdingTransport holds each request until its context ends: before answering, or after answering
// 200 with a body that yields one chunk. ended is closed once the request's context has ended.
type holdingTransport struct {
	midBody bool
	chunk   string
	arrived chan struct{}
	ended   chan struct{}
}

func newHoldingTransport(midBody bool, chunk string) *holdingTransport {
	return &holdingTransport{midBody: midBody, chunk: chunk, arrived: make(chan struct{}), ended: make(chan struct{})}
}

func (tr *holdingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close() // the request body is an in-memory buffer; closing it cannot fail
	}
	close(tr.arrived)
	ctx := req.Context()
	if !tr.midBody {
		<-ctx.Done()
		close(tr.ended)
		return nil, ctx.Err()
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       &holdingBody{ctx: ctx, chunk: []byte(tr.chunk), ended: tr.ended},
		Request:    req,
	}, nil
}

// holdingBody yields its chunk, then blocks until ctx ends and returns ctx's error.
type holdingBody struct {
	ctx   context.Context
	chunk []byte
	ended chan struct{}
	once  sync.Once
}

func (b *holdingBody) Read(p []byte) (int, error) {
	if len(b.chunk) > 0 {
		n := copy(p, b.chunk)
		b.chunk = b.chunk[n:]
		return n, nil
	}
	<-b.ctx.Done()
	b.once.Do(func() { close(b.ended) })
	return 0, b.ctx.Err()
}

func (b *holdingBody) Close() error { return nil }

// signalWriter records what is written to it and closes firstWrite on the first write.
type signalWriter struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	firstWrite chan struct{}
	once       sync.Once
}

func (w *signalWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	n, err := w.buf.Write(p)
	w.mu.Unlock()
	w.once.Do(func() { close(w.firstWrite) })
	return n, err
}

func (w *signalWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// TestTextToSpeechStreamCancelWithHTTPClient verifies that cancelling the context the Client was
// built with cancels a request sent through a supplied client, before its response and mid-body.
func TestTextToSpeechStreamCancelWithHTTPClient(t *testing.T) {
	const chunk = "first chunk"
	testCases := []struct {
		name      string
		midBody   bool
		wantBytes string
	}{
		{name: "cancel before the response", midBody: false, wantBytes: ""},
		{name: "cancel mid-body", midBody: true, wantBytes: chunk},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := countingServer(t, "audio from the server")
			tr := newHoldingTransport(tc.midBody, chunk)
			w := &signalWriter{firstWrite: make(chan struct{})}
			held := tr.arrived
			if tc.midBody {
				held = w.firstWrite
			}

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			client := NewMockClient(ctx, srv.URL, testAPIKey, testRequestTimeout, WithHTTPClient(&http.Client{Transport: tr}))

			errc := make(chan error, 1)
			go func() {
				errc <- client.TextToSpeechStream(w, testVoiceID, TextToSpeechRequest{Text: "hello"})
			}()

			select {
			case <-held:
			case err := <-errc:
				t.Fatalf("TextToSpeechStream() = %v before the supplied transport held the request", err)
			case <-time.After(testWaitBound):
				t.Fatal("timed out waiting for the supplied transport to hold the request")
			}

			cancel()

			select {
			case err := <-errc:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("TextToSpeechStream() = %v, want an error matching %v", err, context.Canceled)
				}
			case <-time.After(testWaitBound):
				t.Fatal("timed out waiting for TextToSpeechStream to return after the cancel")
			}
			select {
			case <-tr.ended:
			default:
				t.Error("the supplied transport never saw its request context end")
			}
			if got := w.String(); got != tc.wantBytes {
				t.Errorf("TextToSpeechStream() wrote %q, want %q", got, tc.wantBytes)
			}
		})
	}
}
