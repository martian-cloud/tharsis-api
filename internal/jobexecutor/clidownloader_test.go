package jobexecutor

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// TestFetchSigningKeysAndVerifyChecksum verifies the happy path:
// fetching signing keys from the well-known endpoint and using them
// to verify a checksum signature. Uses a mock HTTP server.
func TestFetchSigningKeysAndVerifyChecksum(t *testing.T) {
	// Generate a test key pair.
	entity := generateTestKey(t, "hashicorp-test", time.Time{}, 0)

	// Armor the public key (mimics the well-known endpoint response).
	armoredKey := armorEntities(t, entity)

	// Create fake checksum content and sign it with the test key.
	checksumContent := []byte("abc123  terraform_1.12.0_linux_amd64.zip\n")
	signature := signContent(t, entity, checksumContent)

	// Mock server that serves the armored key on the well-known path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(armoredKey)
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	// Step 1: Fetch keys from the mock well-known endpoint.
	keyRing, err := downloader.fetchSigningKeys()
	if err != nil {
		t.Fatalf("fetchSigningKeys failed: %v", err)
	}
	if len(keyRing) == 0 {
		t.Fatal("expected at least one key")
	}
	t.Logf("Fetched %d key(s) from mock endpoint", len(keyRing))

	// Step 2: Verify the signature using the fetched keys.
	err = verifySumsSignature(
		bytes.NewReader(checksumContent),
		bytes.NewReader(signature),
		keyRing,
	)
	if err != nil {
		t.Fatalf("verifySumsSignature failed: %v", err)
	}

	t.Log("Signature verification passed!")
}

// --- Negative scenarios ---

// newTestDownloaderWithURL creates a downloader pointing to a test server URL.
// We do this by using a custom transport that rewrites the well-known URL.
func newTestDownloaderWithURL(t *testing.T, testURL string) *cliDownloader {
	t.Helper()
	return &cliDownloader{
		httpClient: &http.Client{
			Transport: &rewriteTransport{target: testURL},
		},
	}
}

// rewriteTransport redirects any request to the given target URL.
type rewriteTransport struct {
	target string
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	parsed, err := url.Parse(r.target)
	if err != nil {
		return nil, err
	}
	newReq.URL = parsed
	newReq.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(newReq)
}

// TestFetchSigningKeys_EndpointUnreachable verifies the error when
// the well-known endpoint is unreachable.
func TestFetchSigningKeys_EndpointUnreachable(t *testing.T) {
	// Point to an address that won't respond.
	downloader := newTestDownloaderWithURL(t, "http://127.0.0.1:1/does-not-exist")

	_, err := downloader.fetchSigningKeys()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to request signing keys from the well-known endpoint") {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Got expected error: %v", err)
}

// TestFetchSigningKeys_Non200Response verifies the error when
// the well-known endpoint returns a non-200 status code.
func TestFetchSigningKeys_Non200Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	_, err := downloader.fetchSigningKeys()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to fetch signing keys from the well-known endpoint") {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Got expected error: %v", err)
}

// TestFetchSigningKeys_EmptyResponse verifies the error when
// the endpoint returns an empty body.
func TestFetchSigningKeys_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body.
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	_, err := downloader.fetchSigningKeys()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no valid signing keys returned") {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Got expected error: %v", err)
}

// TestFetchSigningKeys_GarbageResponse verifies the error when
// the endpoint returns non-PGP content.
func TestFetchSigningKeys_GarbageResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not a pgp key"))
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	_, err := downloader.fetchSigningKeys()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no valid signing keys returned") {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Got expected error: %v", err)
}

// TestVerifySumsSignature_NoMatchingKey verifies the error when
// the key ring has keys but none match the signature.
func TestVerifySumsSignature_NoMatchingKey(t *testing.T) {
	// Generate a random key that won't match HashiCorp's signature.
	entity, err := openpgp.NewEntity("test", "test key", "test@example.com", nil)
	if err != nil {
		t.Fatalf("failed to create test key: %v", err)
	}
	keyRing := openpgp.EntityList{entity}

	// Use some arbitrary content and a garbage signature.
	checksums := bytes.NewReader([]byte("some checksum content"))
	signature := bytes.NewReader([]byte("not a real signature"))

	err = verifySumsSignature(checksums, signature, keyRing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no signing key matched") {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Got expected error: %v", err)
}

// --- Helpers for multi-key and expired-key tests ---

// generateTestKey creates a fresh PGP key pair for testing.
// If lifetimeSecs > 0, the key will have that lifetime applied.
// If createdAt is non-zero, the key's creation time is set to that value.
func generateTestKey(t *testing.T, name string, createdAt time.Time, lifetimeSecs uint32) *openpgp.Entity {
	t.Helper()
	cfg := &packet.Config{
		DefaultHash:     crypto.SHA256,
		RSABits:         2048,
		KeyLifetimeSecs: lifetimeSecs,
	}
	if !createdAt.IsZero() {
		cfg.Time = func() time.Time { return createdAt }
	}

	entity, err := openpgp.NewEntity(name, "test key", name+"@example.com", cfg)
	if err != nil {
		t.Fatalf("failed to create test key: %v", err)
	}
	return entity
}

// armorEntities serializes one or more PGP entities to ASCII armor format,
// mimicking what the well-known endpoint returns.
func armorEntities(t *testing.T, entities ...*openpgp.Entity) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("failed to create armor encoder: %v", err)
	}
	for _, e := range entities {
		if err := e.Serialize(w); err != nil {
			t.Fatalf("failed to serialize entity: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close armor encoder: %v", err)
	}
	return buf.Bytes()
}

// signContent uses the given entity to produce a detached signature of content.
func signContent(t *testing.T, entity *openpgp.Entity, content []byte) []byte {
	t.Helper()
	var sig bytes.Buffer
	if err := openpgp.DetachSign(&sig, entity, bytes.NewReader(content), nil); err != nil {
		t.Fatalf("failed to sign content: %v", err)
	}
	return sig.Bytes()
}

// TestFetchSigningKeys_MultipleKeys_ShortCircuit verifies that when multiple
// keys are returned from the well-known endpoint, verification stops at the
// first key that successfully verifies the signature.
func TestFetchSigningKeys_MultipleKeys_ShortCircuit(t *testing.T) {
	// Generate 3 distinct keys. Only keyB will sign the content.
	keyA := generateTestKey(t, "keyA", time.Time{}, 0)
	keyB := generateTestKey(t, "keyB", time.Time{}, 0)
	keyC := generateTestKey(t, "keyC", time.Time{}, 0)

	// Serve all 3 keys from the mock well-known endpoint.
	armored := armorEntities(t, keyA, keyB, keyC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(armored)
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	// Step 1: Fetch keys.
	keyRing, err := downloader.fetchSigningKeys()
	if err != nil {
		t.Fatalf("fetchSigningKeys failed: %v", err)
	}
	if len(keyRing) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keyRing))
	}

	// Step 2: Sign content with keyB (the middle key).
	content := []byte("test checksum content")
	signature := signContent(t, keyB, content)

	// Step 3: Verify. verifySumsSignature should find the match on keyB
	// (the 2nd key) and stop.
	err = verifySumsSignature(
		bytes.NewReader(content),
		bytes.NewReader(signature),
		keyRing,
	)
	if err != nil {
		t.Fatalf("expected verification to succeed using keyB, got: %v", err)
	}
	t.Log("Verification succeeded with the matching key (short-circuit behavior confirmed)")
}

// TestFetchSigningKeys_ExpiredKey documents that fetchSigningKeys no longer rejects
// expired keys with a dedicated pre-check (that heuristic was removed because it
// produced false rejections during rotation). Key validity is enforced downstream:
// the OpenPGP library refuses to produce a signature from an expired key, and
// signature verification rejects signatures outside a key's validity window — so an
// expired key cannot validate the checksum file regardless.
func TestFetchSigningKeys_ExpiredKey(t *testing.T) {
	// Create a key "created" 2 years ago with a 1-year lifetime -> already expired.
	twoYearsAgo := time.Now().Add(-2 * 365 * 24 * time.Hour)
	oneYearSeconds := uint32(365 * 24 * 60 * 60)
	expiredKey := generateTestKey(t, "expired", twoYearsAgo, oneYearSeconds)

	armored := armorEntities(t, expiredKey)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(armored)
	}))
	defer server.Close()

	downloader := newTestDownloaderWithURL(t, server.URL)

	keyRing, err := downloader.fetchSigningKeys()
	if err != nil {
		t.Fatalf("fetchSigningKeys should return the parsed key without a pre-check; got error: %v", err)
	}
	if len(keyRing) != 1 {
		t.Fatalf("expected the fetched key to be returned, got %d", len(keyRing))
	}
}

// TestVerifySumsSignature_RejectsExpiredKey is a regression guard for removing the
// explicit expiry pre-check from fetchSigningKeys. It pins the property the production
// code now relies on: signature verification itself rejects a signing key that has
// expired by the time of verification. verifySumsSignature delegates to
// openpgp.CheckDetachedSignature with a nil config (i.e. time.Now()); since a key can't
// sign while expired, we sign with a short-lived key while it is valid and then verify
// with the clock advanced past its lifetime via packet.Config.Time. If a future
// go-crypto upgrade stopped enforcing key expiry at verification, this test would fail
// and signal that the pre-check (or an equivalent) must be reinstated.
func TestVerifySumsSignature_RejectsExpiredKey(t *testing.T) {
	// Key valid now with a 1-hour lifetime, so signing (at time.Now()) succeeds.
	key := generateTestKey(t, "short-lived", time.Time{}, 3600)
	content := []byte("abc123  terraform_1.12.0_linux_amd64.zip\n")
	signature := signContent(t, key, content)
	keyRing := openpgp.EntityList{key}

	// Sanity: it verifies through the production path while the key is valid (nil config = now).
	if err := verifySumsSignature(bytes.NewReader(content), bytes.NewReader(signature), keyRing); err != nil {
		t.Fatalf("expected signature to verify while the key is valid, got: %v", err)
	}

	// With the clock advanced past the key's lifetime, verification must reject it.
	future := &packet.Config{Time: func() time.Time { return time.Now().Add(2 * time.Hour) }}
	_, err := openpgp.CheckDetachedSignature(keyRing, bytes.NewReader(content), bytes.NewReader(signature), future)
	if err == nil {
		t.Fatal("expected signature verification to reject a signing key expired at verification time")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected an expiry error, got: %v", err)
	}
}

// TestFetchSigningKeys_NilSelfSignature verifies that a key with a nil
// SelfSignature does not cause a panic and is treated as expired.
func TestFetchSigningKeys_NilSelfSignature(t *testing.T) {
	entity := generateTestKey(t, "nosig", time.Time{}, 0)

	// Nil out SelfSignature to simulate the edge case.
	for _, id := range entity.Identities {
		id.SelfSignature = nil
	}

	// Directly test that the expiry-check logic does not panic.
	// This mirrors the loop in fetchSigningKeys.
	keyRing := openpgp.EntityList{entity}
	now := time.Now()
	allExpired := true
	for _, e := range keyRing {
		if e.PrimaryKey != nil {
			identity := e.PrimaryIdentity()
			if identity == nil || identity.SelfSignature == nil {
				continue
			}
			lifetime := identity.SelfSignature.KeyLifetimeSecs
			if lifetime == nil || e.PrimaryKey.CreationTime.Add(time.Duration(*lifetime)*time.Second).After(now) {
				allExpired = false
				break
			}
		}
	}

	// Key with nil SelfSignature is skipped, so allExpired stays true.
	if !allExpired {
		t.Fatal("expected allExpired to be true when SelfSignature is nil")
	}
	t.Log("No panic occurred — nil SelfSignature handled safely")
}

func TestSplitArmoredBlocks(t *testing.T) {
	const blockStart = "-----BEGIN PGP PUBLIC KEY BLOCK-----"

	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "empty input",
			input:  "",
			expect: nil,
		},
		{
			name:   "no blocks",
			input:  "just some random text with no markers",
			expect: nil,
		},
		{
			name:   "single block",
			input:  blockStart + "\ndata\n-----END PGP PUBLIC KEY BLOCK-----\n",
			expect: []string{blockStart + "\ndata\n-----END PGP PUBLIC KEY BLOCK-----\n"},
		},
		{
			name:  "two blocks",
			input: blockStart + "\nblock1\n-----END PGP PUBLIC KEY BLOCK-----\n" + blockStart + "\nblock2\n-----END PGP PUBLIC KEY BLOCK-----\n",
			expect: []string{
				blockStart + "\nblock1\n-----END PGP PUBLIC KEY BLOCK-----\n",
				blockStart + "\nblock2\n-----END PGP PUBLIC KEY BLOCK-----\n",
			},
		},
		{
			name:  "three blocks",
			input: blockStart + "\n1\n" + blockStart + "\n2\n" + blockStart + "\n3\n",
			expect: []string{
				blockStart + "\n1\n",
				blockStart + "\n2\n",
				blockStart + "\n3\n",
			},
		},
		{
			name:   "leading text before first block",
			input:  "some preamble text\n" + blockStart + "\ndata\n",
			expect: []string{blockStart + "\ndata\n"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blocks := splitArmoredBlocks(tc.input, blockStart)
			if len(blocks) != len(tc.expect) {
				t.Fatalf("expected %d blocks, got %d", len(tc.expect), len(blocks))
			}
			for i, want := range tc.expect {
				if blocks[i] != want {
					t.Errorf("block %d:\n  got:  %q\n  want: %q", i, blocks[i], want)
				}
			}
		})
	}
}

// downloaderServingKey starts a mock well-known endpoint serving the given armored
// key and returns a downloader pointed at it with the supplied trusted fingerprints.
func downloaderServingKey(t *testing.T, armoredKey []byte, trusted map[string]struct{}) *cliDownloader {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(armoredKey)
	}))
	t.Cleanup(server.Close)
	return &cliDownloader{
		httpClient:          &http.Client{Transport: &rewriteTransport{target: server.URL}},
		trustedFingerprints: trusted,
	}
}

// TestFetchSigningKeys_RejectsUnpinnedFingerprint verifies that a fetched key whose
// primary-key fingerprint is not in the pinned allowlist is rejected, even though it
// parses successfully — this is the protection against a compromised well-known endpoint.
func TestFetchSigningKeys_RejectsUnpinnedFingerprint(t *testing.T) {
	key := generateTestKey(t, "attacker", time.Time{}, 0)
	armored := armorEntities(t, key)

	downloader := downloaderServingKey(t, armored, map[string]struct{}{
		// A fingerprint that does not match the served key.
		"0000000000000000000000000000000000000000": {},
	})

	if _, err := downloader.fetchSigningKeys(); err == nil {
		t.Fatal("expected fetchSigningKeys to reject a key not on the pinned allowlist")
	}
}

// TestFetchSigningKeys_AcceptsPinnedFingerprint verifies that a fetched key whose
// primary-key fingerprint is pinned is accepted.
func TestFetchSigningKeys_AcceptsPinnedFingerprint(t *testing.T) {
	key := generateTestKey(t, "trusted", time.Time{}, 0)
	armored := armorEntities(t, key)
	fp := strings.ToLower(hex.EncodeToString(key.PrimaryKey.Fingerprint))

	downloader := downloaderServingKey(t, armored, map[string]struct{}{fp: {}})

	keyRing, err := downloader.fetchSigningKeys()
	if err != nil {
		t.Fatalf("expected pinned key to be accepted, got error: %v", err)
	}
	if len(keyRing) != 1 {
		t.Fatalf("expected exactly the pinned key, got %d", len(keyRing))
	}
}

// TestFetchSigningKeys_MixedFingerprints verifies that when the endpoint returns
// several keys and only a subset is pinned, exactly the pinned keys are retained and
// the others are dropped. This exercises the in-place keyRing[:0] partial filter where
// a kept key is not the first entity.
func TestFetchSigningKeys_MixedFingerprints(t *testing.T) {
	keyA := generateTestKey(t, "keyA", time.Time{}, 0)
	keyB := generateTestKey(t, "keyB", time.Time{}, 0)
	keyC := generateTestKey(t, "keyC", time.Time{}, 0)
	armored := armorEntities(t, keyA, keyB, keyC)

	// Pin only the middle key.
	fpB := strings.ToLower(hex.EncodeToString(keyB.PrimaryKey.Fingerprint))
	downloader := downloaderServingKey(t, armored, map[string]struct{}{fpB: {}})

	keyRing, err := downloader.fetchSigningKeys()
	if err != nil {
		t.Fatalf("expected the pinned key to be retained, got error: %v", err)
	}
	if len(keyRing) != 1 {
		t.Fatalf("expected only the pinned key to be retained, got %d", len(keyRing))
	}
	gotFP := strings.ToLower(hex.EncodeToString(keyRing[0].PrimaryKey.Fingerprint))
	if gotFP != fpB {
		t.Fatalf("retained key fingerprint = %s, want pinned %s", gotFP, fpB)
	}
}

// TestHashicorpTrustedGPGFingerprints_WellFormed guards the pinned production allowlist
// against typos: each entry must be lowercase hex of a 20-byte v4 fingerprint.
func TestHashicorpTrustedGPGFingerprints_WellFormed(t *testing.T) {
	if len(hashicorpTrustedGPGFingerprints) == 0 {
		t.Fatal("production GPG fingerprint allowlist must not be empty")
	}
	for fp := range hashicorpTrustedGPGFingerprints {
		if fp != strings.ToLower(fp) {
			t.Errorf("fingerprint %q must be lowercase", fp)
		}
		raw, err := hex.DecodeString(fp)
		if err != nil {
			t.Errorf("fingerprint %q is not valid hex: %v", fp, err)
			continue
		}
		if len(raw) != 20 {
			t.Errorf("fingerprint %q decodes to %d bytes, want 20 (v4 fingerprint)", fp, len(raw))
		}
	}
}
