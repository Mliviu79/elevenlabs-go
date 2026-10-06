package elevenlabs

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain verifies that no test in the package leaves a goroutine running once every test has
// finished: a test's server and idle connections are closed in its own cleanup, never ignored.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
