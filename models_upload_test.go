package elevenlabs

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"
	"testing/iotest"
)

// recordingContent is the content of an upload: it reads from its Reader, counts how often it
// was closed and reports closeErr from every close.
type recordingContent struct {
	io.Reader
	closed   int
	closeErr error
}

func (c *recordingContent) Close() error {
	c.closed++
	return c.closeErr
}

// TestWriteFilePart verifies that the upload part writer writes the content as a part of the
// files field, closes the content exactly once on every path, and reports a close failure, also
// behind an earlier one.
func TestWriteFilePart(t *testing.T) {
	errRead := errors.New("content read failed")

	t.Run("writes_the_part_and_closes_the_content", func(t *testing.T) {
		const fileName, fileContent = "sample.mp3", "sample audio"
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		content := &recordingContent{Reader: strings.NewReader(fileContent)}

		if err := writeFilePart(w, fileName, content); err != nil {
			t.Fatalf("writeFilePart() = %v, want nil", err)
		}

		if content.closed != 1 {
			t.Errorf("content was closed %d times, want 1", content.closed)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("closing the multipart writer: %v", err)
		}
		part, err := multipart.NewReader(&body, w.Boundary()).NextPart()
		if err != nil {
			t.Fatalf("reading the part back: %v", err)
		}
		got, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("reading the part's content back: %v", err)
		}
		if part.FormName() != "files" || part.FileName() != fileName || string(got) != fileContent {
			t.Errorf("part = field %q, file %q, content %q; want field %q, file %q, content %q",
				part.FormName(), part.FileName(), got, "files", fileName, fileContent)
		}
	})

	t.Run("reports_a_close_failure", func(t *testing.T) {
		w := multipart.NewWriter(&bytes.Buffer{})
		content := &recordingContent{Reader: strings.NewReader("sample audio"), closeErr: errClose}

		err := writeFilePart(w, "sample.mp3", content)

		if !errors.Is(err, errClose) {
			t.Errorf("writeFilePart() = %v, want it to match the content's close failure", err)
		}
		if content.closed != 1 {
			t.Errorf("content was closed %d times, want 1", content.closed)
		}
	})

	t.Run("reports_a_close_failure_behind_a_copy_failure", func(t *testing.T) {
		w := multipart.NewWriter(&bytes.Buffer{})
		content := &recordingContent{Reader: iotest.ErrReader(errRead), closeErr: errClose}

		err := writeFilePart(w, "sample.mp3", content)

		if !errors.Is(err, errRead) {
			t.Errorf("writeFilePart() = %v, want it to match the content's read failure", err)
		}
		if !errors.Is(err, errClose) {
			t.Errorf("writeFilePart() = %v, want it to match the content's close failure", err)
		}
		if content.closed != 1 {
			t.Errorf("content was closed %d times, want 1", content.closed)
		}
	})
}
