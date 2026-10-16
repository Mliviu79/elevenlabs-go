package elevenlabs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	errClose = errors.New("close failed")
	errWrite = errors.New("write failed")
)

// closeFailRT sends each request over the default transport and wraps the response body so that
// Close closes the real body and then reports errClose.
type closeFailRT struct{}

func (closeFailRT) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = closeFailBody{resp.Body}
	return resp, nil
}

type closeFailBody struct{ io.ReadCloser }

func (b closeFailBody) Close() error {
	_ = b.ReadCloser.Close() // the real body's own close result is not what these tests observe
	return errClose
}

// failingWriter fails its first write with errWrite.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func closeFailClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewMockClient(srv.URL, testAPIKey, testRequestTimeout, WithHTTPClient(&http.Client{Transport: closeFailRT{}}))
}

func assertBodyCloseError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, errClose) {
		t.Errorf("error = %v, want it to match the body's close failure", err)
	}
	var ce *BodyCloseError
	if !errors.As(err, &ce) {
		t.Errorf("error = %v, want it to carry a *BodyCloseError", err)
	}
}

// TestDoRequest_CloseFailureAfterSuccessIsReturned verifies that a request whose body was copied in
// full still reports the body's close failure.
func TestDoRequest_CloseFailureAfterSuccessIsReturned(t *testing.T) {
	const body = `{"audio_base64":"AA=="}`
	client := closeFailClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	})
	var got signalWriter
	got.firstWrite = make(chan struct{})

	err := client.TextToSpeechStreamWithTimestamps(t.Context(), &got, testVoiceID, TextToSpeechRequest{Text: "hello"})

	assertBodyCloseError(t, err)
	if got.String() != body {
		t.Errorf("wrote %q, want the whole body %q", got.String(), body)
	}
}

// TestDoRequest_CloseFailureJoinsCancellation verifies that a request cancelled mid-body reports
// both the cancellation and the body's close failure, and returns promptly.
func TestDoRequest_CloseFailureJoinsCancellation(t *testing.T) {
	client := closeFailClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"audio_base64":"AA=="}`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := &signalWriter{firstWrite: make(chan struct{})}
	errc := make(chan error, 1)
	go func() {
		errc <- client.TextToSpeechStreamWithTimestamps(ctx, w, testVoiceID, TextToSpeechRequest{Text: "hello"})
	}()

	select {
	case <-w.firstWrite:
	case err := <-errc:
		t.Fatalf("request returned %v before the first object arrived", err)
	case <-time.After(testWaitBound):
		t.Fatal("timed out waiting for the first object")
	}
	cancel()

	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want it to match context.Canceled", err)
		}
		assertBodyCloseError(t, err)
	case <-time.After(testWaitBound):
		t.Fatal("the cancelled request did not return")
	}
}

// TestDoRequest_CloseFailureJoinsWriteError verifies that a failure to write the body and the
// body's close failure are both reported.
func TestDoRequest_CloseFailureJoinsWriteError(t *testing.T) {
	client := closeFailClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"audio_base64":"AA=="}`)
	})

	err := client.TextToSpeechStreamWithTimestamps(t.Context(), failingWriter{}, testVoiceID, TextToSpeechRequest{Text: "hello"})

	if !errors.Is(err, errWrite) {
		t.Errorf("error = %v, want it to match the write failure", err)
	}
	assertBodyCloseError(t, err)
}

// TestDoRequest_CloseFailureJoinsStatusError verifies that an unsuccessful status and the body's
// close failure are both reported.
func TestDoRequest_CloseFailureJoinsStatusError(t *testing.T) {
	client := closeFailClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := client.GetModels(t.Context())

	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("error = %v, want a *StatusError with StatusCode %d", err, http.StatusServiceUnavailable)
	}
	assertBodyCloseError(t, err)
}
