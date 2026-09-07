package cloudflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TODO(§7): ListZones itself is not unit tested here — same reason as the
// AWS and Azure connectors: it depends directly on the Cloudflare SDK's
// concrete client and pagination types, which need either a live account or
// an HTTP-fixture recording/replay harness to test properly. Verified
// manually against a real account via cmd/live-cloudflare-check.

func TestTokenFromFile(t *testing.T) {
	t.Run("reads and trims a token file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		if err := os.WriteFile(path, []byte("  my-test-token\n"), 0o600); err != nil {
			t.Fatalf("writing test fixture: %v", err)
		}

		got, err := TokenFromFile(path)
		if err != nil {
			t.Fatalf("TokenFromFile() error = %v", err)
		}
		if got != "my-test-token" {
			t.Errorf("TokenFromFile() = %q, want %q", got, "my-test-token")
		}
	})

	t.Run("missing file returns an actionable error, not a bare not-found", func(t *testing.T) {
		_, err := TokenFromFile(filepath.Join(t.TempDir(), "does-not-exist"))
		if err == nil {
			t.Fatal("expected an error for a missing token file, got nil")
		}
		if !strings.Contains(err.Error(), "dash.cloudflare.com") {
			t.Errorf("error should point to where to generate a token, got: %v", err)
		}
	})
}
