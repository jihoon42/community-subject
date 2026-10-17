package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	src := []byte("---\nid: gauss_jordan\ntitle: \"Gauss\"\nversion: 0.1\ntags: [linear-algebra, c]\nempty: []\n---\n# Body\n\ntext\n")
	fm, body, err := splitFrontmatter(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := fm.Get("id"); got != "gauss_jordan" {
		t.Errorf("id = %q", got)
	}
	if got := fm.Get("title"); got != "Gauss" {
		t.Errorf("title = %q, quotes should be stripped", got)
	}
	if got := fm.Get("version"); got != "0.1" {
		t.Errorf("version = %q", got)
	}
	if got := fm.Get("tags"); got != "linear-algebra, c" {
		t.Errorf("tags = %q", got)
	}
	var tags []string
	for _, f := range fm.Fields {
		if f.Key == "tags" {
			tags = f.List
		}
	}
	if len(tags) != 2 || tags[0] != "linear-algebra" || tags[1] != "c" {
		t.Errorf("tags list = %v", tags)
	}
	if fm.Get("missing") != "" {
		t.Error("missing key should be empty")
	}
	if string(body) != "# Body\n\ntext\n" {
		t.Errorf("body = %q", body)
	}
	if len(fm.Fields) != 5 || fm.Fields[0].Key != "id" || fm.Fields[4].Key != "empty" {
		t.Errorf("field order lost: %+v", fm.Fields)
	}
}

func TestSplitFrontmatterErrors(t *testing.T) {
	if _, _, err := splitFrontmatter([]byte("# no frontmatter\n")); err == nil {
		t.Error("expected error when --- is missing")
	}
	if _, _, err := splitFrontmatter([]byte("---\nid: x\n")); err == nil {
		t.Error("expected error when the closing --- is missing")
	}
}

func TestRealSubjectFrontmatter(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "subjects", "gauss_jordan", "SUBJECT.md"))
	if err != nil {
		t.Skip("subject file not found:", err)
	}
	fm, body, err := splitFrontmatter(src)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Get("id") != "gauss_jordan" || fm.Get("version") != "0.1" {
		t.Errorf("unexpected id/version: %q %q", fm.Get("id"), fm.Get("version"))
	}
	if !strings.HasPrefix(string(body), "\n# gauss_jordan") && !strings.HasPrefix(string(body), "# gauss_jordan") {
		t.Errorf("body should start with the title heading, got %q", body[:40])
	}
}

// signedFixture creates a throwaway ed25519 key, a record, its signature
// and an allowed_signers file in dir. It returns the record path.
func signedFixture(t *testing.T, dir, identity, content string) string {
	t.Helper()
	key := filepath.Join(dir, "key")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", identity, "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	record := filepath.Join(dir, "record.json")
	if err := os.WriteFile(record, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("ssh-keygen", "-Y", "sign", "-f", key, "-n", namespace, record).CombinedOutput(); err != nil {
		t.Fatalf("sign: %v\n%s", err, out)
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(pub))
	line := identity + " namespaces=\"" + namespace + "\" " + fields[0] + " " + fields[1] + "\n"
	if err := os.WriteFile(filepath.Join(dir, "allowed_signers"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	return record
}

func newVerifier(t *testing.T, dir string) Verifier {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "..", "kit", "record", "verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skip("verify.sh not found:", err)
	}
	return Verifier{Script: script, AllowedSigners: filepath.Join(dir, "allowed_signers")}
}

func TestVerifyOK(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	dir := t.TempDir()
	record := signedFixture(t, dir, "tester@example", `{"kind":"human","flag":"Ok"}`)
	v := newVerifier(t, dir)
	got := v.Verify(record)
	if !got.OK {
		t.Fatalf("expected OK, got %+v", got)
	}
	if got.Signer != "tester@example" {
		t.Errorf("signer = %q", got.Signer)
	}
}

func TestVerifyFailAfterTamper(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	dir := t.TempDir()
	record := signedFixture(t, dir, "tester@example", `{"kind":"human","flag":"Ok"}`)
	v := newVerifier(t, dir)
	// Change one byte of the record: the signature must no longer match.
	if err := os.WriteFile(record, []byte(`{"kind":"human","flag":"Ko"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := v.Verify(record)
	if got.OK {
		t.Fatalf("expected FAIL after tampering, got %+v", got)
	}
	if got.Signer != "tester@example" {
		t.Errorf("signer should still be found from the key, got %q", got.Signer)
	}
}

func TestVerifyFailUnknownKey(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	dir := t.TempDir()
	record := signedFixture(t, dir, "tester@example", `{"kind":"human","flag":"Ok"}`)
	v := newVerifier(t, dir)
	// An allowed_signers file that does not list the key.
	if err := os.WriteFile(v.AllowedSigners, []byte("nobody@example namespaces=\"community-subject\" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIfq/UmGU/8ndI1P8MMIE+o7dpIi7msSPq0qhNe83FQ\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := v.Verify(record)
	if got.OK || got.Signer != "" {
		t.Fatalf("expected FAIL with unknown signer, got %+v", got)
	}
	// No signature file at all.
	os.Remove(record + ".sig")
	if got := v.Verify(record); got.OK {
		t.Fatalf("expected FAIL without .sig, got %+v", got)
	}
}

func TestLoadRecordEvaluatorMismatch(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	dir := t.TempDir()
	record := signedFixture(t, dir, "tester@example",
		`{"kind":"human","subject":{"id":"s","version":"0.1"},"evaluator":"someone-else@example","flag":"Ok","date":"2026-10-18"}`)
	v := newVerifier(t, dir)
	r, err := loadRecord(record, "owner", v)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Error("record whose evaluator differs from the signer must not be OK")
	}
	if r.Kind != "human" || r.Flag != "Ok" {
		t.Errorf("record fields: %+v", r)
	}
}

func TestBuildSite(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "subjects")); err != nil {
		t.Skip("repository root not found")
	}
	out := filepath.Join(t.TempDir(), "public")
	files, err := build(root, out)
	if err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>", "과제 목록", "subjects/gauss_jordan/index.html", "style.css"} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html lacks %q", want)
		}
	}
	subject, err := os.ReadFile(filepath.Join(out, "subjects", "gauss_jordan", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<table", "Telecadeta", "기계 평가 키트", "I. 머리말</h2>"} {
		if !strings.Contains(string(subject), want) {
			t.Errorf("subject page lacks %q", want)
		}
	}
	if len(files) < 3 {
		t.Errorf("expected at least 3 files, got %v", files)
	}
}
