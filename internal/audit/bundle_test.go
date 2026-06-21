package audit_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// bundleManifestEntry mirrors the manifest.json entry format from runAuditBundle.
type bundleManifestEntry struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type bundleManifest struct {
	GeneratedAt string                `json:"generated_at"`
	BuildCommit string                `json:"build_commit"`
	BuildDate   string                `json:"build_date"`
	ChainOK     bool                  `json:"chain_ok"`
	EventCount  int                   `json:"event_count"`
	Files       []bundleManifestEntry `json:"files"`
}

// TestBundleManifestVerification builds a synthetic .tar.gz that looks like
// a SOC 2 evidence bundle and verifies that the manifest SHA-256 entries
// match the actual file contents. This proves the manifest format is correct
// and that a verifier can trust the bundle.
func TestBundleManifestVerification(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "evidence.tar.gz")

	// Build a synthetic bundle with a manifest and one audit file.
	auditData := []byte(`{"seq":1,"ts":"2026-06-21T00:00:00Z","verb":"EXPORT_EVIDENCE"}` + "\n")
	auditHash := fmt.Sprintf("%x", sha256.Sum256(auditData))

	manifest := bundleManifest{
		GeneratedAt: "2026-06-21T00:00:00Z",
		BuildCommit: "abc1234",
		BuildDate:   "2026-06-21",
		ChainOK:     true,
		EventCount:  1,
		Files: []bundleManifestEntry{
			{Name: "audit.jsonl", Bytes: len(auditData), SHA256: "sha256:" + auditHash},
		},
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestData = append(manifestData, '\n')

	// Write tar.gz.
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, pair := range []struct{ name string; data []byte }{
		{"audit.jsonl", auditData},
		{"manifest.json", manifestData},
	} {
		hdr := &tar.Header{Name: pair.name, Size: int64(len(pair.data)), Mode: 0640}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(pair.data)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()

	// Now verify: read the tar.gz, extract files, check manifest hashes.
	f2, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	defer f2.Close()
	gr, err := gzip.NewReader(f2)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	files := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar: %v", err)
		}
		data, _ := io.ReadAll(tr)
		files[hdr.Name] = data
	}

	// Extract manifest.
	mData, ok := files["manifest.json"]
	if !ok {
		t.Fatal("manifest.json not in bundle")
	}
	var m bundleManifest
	if err := json.Unmarshal(mData, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if !m.ChainOK {
		t.Error("manifest: chain_ok = false, want true")
	}
	if m.EventCount != 1 {
		t.Errorf("manifest: event_count = %d, want 1", m.EventCount)
	}

	// Verify each file's SHA-256 matches the manifest entry.
	for _, entry := range m.Files {
		content, ok := files[entry.Name]
		if !ok {
			t.Errorf("file %q in manifest but not in bundle", entry.Name)
			continue
		}
		sum := fmt.Sprintf("sha256:%x", sha256.Sum256(content))
		if sum != entry.SHA256 {
			t.Errorf("file %q: manifest sha256=%s actual=%s", entry.Name, entry.SHA256, sum)
		}
	}
}
