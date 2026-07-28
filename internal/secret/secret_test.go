package secret

import (
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/vinz"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
)

// mockVinz is a minimal VinzServicer that records whether SaveKey was called.
type mockVinz struct {
	getKey     vinz.Key
	getErr     error
	saveCalled bool
	saveErr    error
}

func (m *mockVinz) GetKey(organizationID, projectIdOrAlternateId string) (vinz.Key, error) {
	return m.getKey, m.getErr
}

func (m *mockVinz) SaveKey(organizationID, projectIdOrAlternateId string, key vinz.Key) (vinz.Key, error) {
	m.saveCalled = true
	return key, m.saveErr
}

// withMockVinz installs a mock Vinz service, runs the test body in an empty
// working directory (so no stray local .hxkey is found), and restores state.
func withMockVinz(t *testing.T, mock *mockVinz) {
	t.Helper()

	prev := vs
	vs = mock
	t.Cleanup(func() { vs = prev })

	prevLocal := flags.LocalSecret
	flags.LocalSecret = false
	t.Cleanup(func() { flags.LocalSecret = prevLocal })

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

// A 401 from Vinz must never be mistaken for "no secret exists" and must never
// result in a new key being generated/saved — that is what orphaned prior keys.
func TestLoadOrInitializeSecret_DoesNotSaveOnUnauthorized(t *testing.T) {
	mock := &mockVinz{getErr: errors.Wrapf(errors.ErrUnauthorized, "unauthorized")}
	withMockVinz(t, mock)

	_, _, err := LoadOrInitializeSecret("org_test", "proj_test")

	if err == nil {
		t.Fatal("expected an error when Vinz returns 401, got nil")
	}
	if !errors.Is(err, errors.ErrUnauthorized) {
		t.Fatalf("expected the underlying 401 to be surfaced, got: %v", err)
	}
	if mock.saveCalled {
		t.Fatal("SaveKey must NOT be called after a 401 — a new key would orphan the real one")
	}
}

// The same guard applies to any non-404 failure (e.g. a 5xx / transient error).
func TestLoadOrInitializeSecret_DoesNotSaveOnServerError(t *testing.T) {
	mock := &mockVinz{getErr: errors.Wrapf(errors.ErrInternalServerError, "internal server error")}
	withMockVinz(t, mock)

	_, _, err := LoadOrInitializeSecret("org_test", "proj_test")

	if err == nil {
		t.Fatal("expected an error when Vinz returns 500, got nil")
	}
	if mock.saveCalled {
		t.Fatal("SaveKey must NOT be called after a 500")
	}
}

// A genuine 404 means the key truly does not exist yet, so initialization
// (generate + SaveKey) is the correct behavior and must still happen.
func TestLoadOrInitializeSecret_GeneratesOnNotFound(t *testing.T) {
	mock := &mockVinz{getErr: errors.Wrapf(errors.ErrNotFound, "not found")}
	withMockVinz(t, mock)

	secret, location, err := LoadOrInitializeSecret("org_test", "proj_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location != SecretLocationVinz {
		t.Fatalf("expected SecretLocationVinz, got %v", location)
	}
	if !mock.saveCalled {
		t.Fatal("SaveKey must be called to persist the newly generated key on a real 404")
	}
	if secret.Base64SecretKey == "" {
		t.Fatal("expected a generated secret to be returned")
	}
}

// LoadSecret itself must surface a 401 as an error rather than an empty
// SecretLocationNone that callers would misread as "no key".
func TestLoadSecret_SurfacesUnauthorized(t *testing.T) {
	mock := &mockVinz{getErr: errors.Wrapf(errors.ErrUnauthorized, "unauthorized")}
	withMockVinz(t, mock)

	_, location, err := LoadSecret("org_test", "proj_test")

	if err == nil {
		t.Fatal("expected LoadSecret to surface the 401 error")
	}
	if !errors.Is(err, errors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
	if location != SecretLocationNone {
		t.Fatalf("expected SecretLocationNone, got %v", location)
	}
}

// A real 404 with no local fallback yields SecretLocationNone and no error.
func TestLoadSecret_NotFoundReturnsNoneWithoutError(t *testing.T) {
	mock := &mockVinz{getErr: errors.Wrapf(errors.ErrNotFound, "not found")}
	withMockVinz(t, mock)

	_, location, err := LoadSecret("org_test", "proj_test")

	if err != nil {
		t.Fatalf("expected no error on a genuine 404, got: %v", err)
	}
	if location != SecretLocationNone {
		t.Fatalf("expected SecretLocationNone, got %v", location)
	}
}
