package elevenlabs

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

// stubTransport answers every request with resp and err, as given.
type stubTransport struct {
	resp *http.Response
	err  error
}

func (tr stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close() // the request body is an in-memory buffer; closing it cannot fail
	}
	return tr.resp, tr.err
}

// assertStepNamed verifies that err's text opens with step, the action that failed.
func assertStepNamed(t *testing.T, err error, step string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want one that opens with the failed step %q", step)
	}
	if !strings.HasPrefix(err.Error(), step+": ") {
		t.Errorf("error = %q, want it to open with the failed step %q", err, step)
	}
}

// TestDoRequest_FailedStepIsNamed verifies, for each step of a request that can fail outside the
// status dispatch, that the returned error still matches its cause and names the step in front.
func TestDoRequest_FailedStepIsNamed(t *testing.T) {
	errSend := errors.New("transport refused the request")
	errRead := errors.New("body read failed")

	t.Run("build_request", func(t *testing.T) {
		tr := &recordingTransport{}
		client := NewMockClient(unservedBaseURL, testAPIKey, testRequestTimeout, WithHTTPClient(&http.Client{Transport: tr}))

		// A control character makes the request URL unparseable.
		_, err := client.TextToSpeech(t.Context(), "voice\nid", TextToSpeechRequest{Text: "hello"})

		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			t.Errorf("error = %v, want it to carry the *url.Error of the URL that did not parse", err)
		}
		if served := tr.served(); len(served) != 0 {
			t.Errorf("transport served %d requests, want none: the request could not be built", len(served))
		}
		assertStepNamed(t, err, "build request")
	})

	t.Run("send_request", func(t *testing.T) {
		hc := &http.Client{Transport: stubTransport{err: errSend}}
		client := NewMockClient(unservedBaseURL, testAPIKey, testRequestTimeout, WithHTTPClient(hc))

		_, err := client.TextToSpeech(t.Context(), testVoiceID, TextToSpeechRequest{Text: "hello"})

		if !errors.Is(err, errSend) {
			t.Errorf("error = %v, want it to match the transport's failure", err)
		}
		assertStepNamed(t, err, "send request")
	})

	t.Run("read_response_body", func(t *testing.T) {
		hc := &http.Client{Transport: stubTransport{resp: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Status:     "503 Service Unavailable",
			Header:     http.Header{},
			Body:       io.NopCloser(iotest.ErrReader(errRead)),
		}}}
		client := NewMockClient(unservedBaseURL, testAPIKey, testRequestTimeout, WithHTTPClient(hc))

		_, err := client.TextToSpeech(t.Context(), testVoiceID, TextToSpeechRequest{Text: "hello"})

		if !errors.Is(err, errRead) {
			t.Errorf("error = %v, want it to match the body's read failure", err)
		}
		assertStepNamed(t, err, "read response body")
	})

	t.Run("copy_response_body", func(t *testing.T) {
		hc := &http.Client{Transport: &recordingTransport{body: "audio"}}
		client := NewMockClient(unservedBaseURL, testAPIKey, testRequestTimeout, WithHTTPClient(hc))

		err := client.TextToSpeechStream(t.Context(), failingWriter{}, testVoiceID, TextToSpeechRequest{Text: "hello"})

		if !errors.Is(err, errWrite) {
			t.Errorf("error = %v, want it to match the write failure", err)
		}
		assertStepNamed(t, err, "copy response body")
	})
}
