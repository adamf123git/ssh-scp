package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

// ---------------------------------------------------------------------------
// shellQuote
// ---------------------------------------------------------------------------

func TestShellQuoteSimple(t *testing.T) {
	got := shellQuote("/home/user/file.txt")
	want := "'/home/user/file.txt'"
	if got != want {
		t.Errorf("shellQuote simple = %q, want %q", got, want)
	}
}

func TestShellQuoteWithSingleQuote(t *testing.T) {
	got := shellQuote("/home/user/it's a file")
	want := "'/home/user/it'\\''s a file'"
	if got != want {
		t.Errorf("shellQuote with quote = %q, want %q", got, want)
	}
}

func TestShellQuoteEmpty(t *testing.T) {
	got := shellQuote("")
	want := "''"
	if got != want {
		t.Errorf("shellQuote empty = %q, want %q", got, want)
	}
}

func TestShellQuoteSpaces(t *testing.T) {
	got := shellQuote("/path/with spaces")
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("shellQuote should wrap in single quotes, got %q", got)
	}
}

func TestShellQuoteMultipleSingleQuotes(t *testing.T) {
	got := shellQuote("it's a 'test' path")
	if strings.Count(got, `'\''`) != 3 {
		t.Errorf("should escape 3 single quotes, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// splitLines
// ---------------------------------------------------------------------------

func TestSplitLinesMultiple(t *testing.T) {
	got := splitLines("line1\nline2\nline3")
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}
	if got[0] != "line1" || got[1] != "line2" || got[2] != "line3" {
		t.Errorf("unexpected lines: %v", got)
	}
}

func TestSplitLinesTrailingNewline(t *testing.T) {
	got := splitLines("line1\nline2\n")
	if len(got) != 2 {
		t.Fatalf("expected 2 lines (trailing newline stripped), got %d: %v", len(got), got)
	}
}

func TestSplitLinesEmpty(t *testing.T) {
	got := splitLines("")
	if len(got) != 0 {
		t.Errorf("expected 0 lines, got %d", len(got))
	}
}

func TestSplitLinesSingle(t *testing.T) {
	got := splitLines("hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("expected [hello], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// splitFields
// ---------------------------------------------------------------------------

func TestSplitFieldsSpaces(t *testing.T) {
	got := splitFields("  a   b   c  ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("splitFields with spaces = %v", got)
	}
}

func TestSplitFieldsTabs(t *testing.T) {
	got := splitFields("a\tb\tc")
	if len(got) != 3 {
		t.Errorf("splitFields with tabs, expected 3, got %d: %v", len(got), got)
	}
}

func TestSplitFieldsEmpty(t *testing.T) {
	got := splitFields("")
	if len(got) != 0 {
		t.Errorf("splitFields empty, expected 0, got %d", len(got))
	}
}

func TestSplitFieldsSingle(t *testing.T) {
	got := splitFields("hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("splitFields single = %v", got)
	}
}

// ---------------------------------------------------------------------------
// parsePerm
// ---------------------------------------------------------------------------

func TestParsePermFull(t *testing.T) {
	mode := parsePerm("-rwxrwxrwx")
	if mode != 0777 {
		t.Errorf("parsePerm -rwxrwxrwx = %o, want 0777", mode)
	}
}

func TestParsePermReadOnly(t *testing.T) {
	mode := parsePerm("-r--r--r--")
	if mode != 0444 {
		t.Errorf("parsePerm -r--r--r-- = %o, want 0444", mode)
	}
}

func TestParsePermOwnerOnly(t *testing.T) {
	mode := parsePerm("-rwx------")
	if mode != 0700 {
		t.Errorf("parsePerm -rwx------ = %o, want 0700", mode)
	}
}

func TestParsePermDirectory(t *testing.T) {
	mode := parsePerm("drwxr-xr-x")
	if mode != 0755 {
		t.Errorf("parsePerm drwxr-xr-x = %o, want 0755", mode)
	}
}

func TestParsePermShortString(t *testing.T) {
	mode := parsePerm("short")
	if mode != 0 {
		t.Errorf("parsePerm short input = %o, want 0", mode)
	}
}

func TestParsePermEmpty(t *testing.T) {
	mode := parsePerm("")
	if mode != 0 {
		t.Errorf("parsePerm empty = %o, want 0", mode)
	}
}

func TestParsePermNoExec(t *testing.T) {
	mode := parsePerm("-rw-rw-rw-")
	if mode != 0666 {
		t.Errorf("parsePerm -rw-rw-rw- = %o, want 0666", mode)
	}
}

func TestParsePermTypical600(t *testing.T) {
	mode := parsePerm("-rw-------")
	if mode != 0600 {
		t.Errorf("parsePerm -rw------- = %o, want 0600", mode)
	}
}

// ---------------------------------------------------------------------------
// parseLSLine
// ---------------------------------------------------------------------------

func TestParseLSLineRegularFile(t *testing.T) {
	line := "-rw-r--r-- 1 user group 12345 2024-01-15 10:30:00 myfile.txt"
	f := parseLSLine(line)
	if f == nil {
		t.Fatal("parseLSLine returned nil for valid line")
	}
	if f.Name != "myfile.txt" {
		t.Errorf("Name = %q, want %q", f.Name, "myfile.txt")
	}
	if f.Size != 12345 {
		t.Errorf("Size = %d, want 12345", f.Size)
	}
	if f.IsDir {
		t.Error("should not be dir")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %o, want 0644", f.Mode)
	}
}

func TestParseLSLineDirectory(t *testing.T) {
	line := "drwxr-xr-x 2 user group 4096 2024-01-15 10:30:00 mydir"
	f := parseLSLine(line)
	if f == nil {
		t.Fatal("parseLSLine returned nil for dir line")
	}
	if f.Name != "mydir" {
		t.Errorf("Name = %q, want %q", f.Name, "mydir")
	}
	if !f.IsDir {
		t.Error("should be dir")
	}
}

func TestParseLSLineShort(t *testing.T) {
	f := parseLSLine("abc")
	if f != nil {
		t.Errorf("expected nil for short line, got %+v", f)
	}
}

func TestParseLSLineEmpty(t *testing.T) {
	f := parseLSLine("")
	if f != nil {
		t.Errorf("expected nil for empty line, got %+v", f)
	}
}

func TestParseLSLineFiveFields(t *testing.T) {
	// Some ls formats have fewer fields
	line := "-rw-r--r-- 1 user group readme.txt"
	f := parseLSLine(line)
	if f == nil {
		t.Fatal("should parse 5-field line")
	}
	if f.Name != "readme.txt" {
		t.Errorf("Name = %q, want %q", f.Name, "readme.txt")
	}
}

// ---------------------------------------------------------------------------
// parseLS
// ---------------------------------------------------------------------------

func TestParseLSBasic(t *testing.T) {
	output := `total 16
drwxr-xr-x 2 user user 4096 2024-01-15 10:00:00 .
drwxr-xr-x 3 user user 4096 2024-01-15 10:00:00 ..
-rw-r--r-- 1 user user 1024 2024-01-15 10:00:00 file1.txt
-rw-r--r-- 1 user user 2048 2024-01-15 10:00:00 file2.txt
drwxr-xr-x 2 user user 4096 2024-01-15 10:00:00 subdir
`
	files := parseLS(output)
	if len(files) != 3 {
		t.Fatalf("expected 3 files (excluding . and ..), got %d", len(files))
	}

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	for _, n := range []string{"file1.txt", "file2.txt", "subdir"} {
		if !names[n] {
			t.Errorf("missing file %q", n)
		}
	}
}

func TestParseLSEmpty(t *testing.T) {
	files := parseLS("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParseLSTotalOnly(t *testing.T) {
	files := parseLS("total 0\n")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParseLSSkipsDotEntries(t *testing.T) {
	output := "drwxr-xr-x 2 u g 4096 2024-01-15 10:00:00 .\ndrwxr-xr-x 2 u g 4096 2024-01-15 10:00:00 ..\n"
	files := parseLS(output)
	if len(files) != 0 {
		t.Errorf("expected 0 after filtering . and .., got %d", len(files))
	}
}

// ---------------------------------------------------------------------------
// PasswordAuth
// ---------------------------------------------------------------------------

func TestPasswordAuth(t *testing.T) {
	auth := PasswordAuth("secret")
	if auth == nil {
		t.Fatal("PasswordAuth returned nil")
	}
}

// ---------------------------------------------------------------------------
// PubKeyAuth
// ---------------------------------------------------------------------------

func TestPubKeyAuthNonexistentFile(t *testing.T) {
	_, err := PubKeyAuth("/nonexistent/path/key")
	if err == nil {
		t.Error("expected error for nonexistent key file")
	}
}

func TestPubKeyAuthInvalidKeyContent(t *testing.T) {
	tmp := t.TempDir()
	keyFile := tmp + "/badkey"
	if err := os.WriteFile(keyFile, []byte("not a real key"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := PubKeyAuth(keyFile)
	if err == nil {
		t.Error("expected error for invalid key content")
	}
}

// ---------------------------------------------------------------------------
// RemoteFile struct
// ---------------------------------------------------------------------------

func TestRemoteFileFields(t *testing.T) {
	f := RemoteFile{
		Name:  "test.txt",
		Size:  1024,
		IsDir: false,
	}
	if f.Name != "test.txt" || f.Size != 1024 || f.IsDir {
		t.Errorf("RemoteFile fields not set correctly: %+v", f)
	}
}

// ---------------------------------------------------------------------------
// parseAlgorithms
// ---------------------------------------------------------------------------

func TestParseAlgorithmsEmpty(t *testing.T) {
	got := parseAlgorithms("")
	if got != nil {
		t.Errorf("parseAlgorithms(\"\") = %v, want nil", got)
	}
}

func TestParseAlgorithmsPlain(t *testing.T) {
	got := parseAlgorithms("ssh-ed25519,rsa-sha2-256")
	if len(got) != 2 {
		t.Fatalf("expected 2 algos, got %d: %v", len(got), got)
	}
	if got[0] != "ssh-ed25519" || got[1] != "rsa-sha2-256" {
		t.Errorf("unexpected algos: %v", got)
	}
}

func TestParseAlgorithmsAppendPrefix(t *testing.T) {
	got := parseAlgorithms("+ssh-rsa")
	// Should prepend defaults and then append ssh-rsa
	if len(got) < 2 {
		t.Fatalf("expected defaults + appended, got %d: %v", len(got), got)
	}
	// The last element should be the appended one
	if got[len(got)-1] != "ssh-rsa" {
		t.Errorf("last algo = %q, want %q", got[len(got)-1], "ssh-rsa")
	}
	// First should be a default (ssh-ed25519)
	if got[0] != "ssh-ed25519" {
		t.Errorf("first algo = %q, want %q", got[0], "ssh-ed25519")
	}
}

func TestParseAlgorithmsWhitespace(t *testing.T) {
	got := parseAlgorithms("  ssh-ed25519 , rsa-sha2-256  ")
	if len(got) != 2 {
		t.Fatalf("expected 2 algos, got %d: %v", len(got), got)
	}
	if got[0] != "ssh-ed25519" || got[1] != "rsa-sha2-256" {
		t.Errorf("unexpected algos: %v", got)
	}
}

func TestParseAlgorithmsEmptyElements(t *testing.T) {
	got := parseAlgorithms("ssh-ed25519,,rsa-sha2-256")
	// empty element between commas should be skipped
	if len(got) != 2 {
		t.Fatalf("expected 2 algos, got %d: %v", len(got), got)
	}
}

// ---------------------------------------------------------------------------
// PasswordCallbackAuth / KeyboardInteractiveAuth / SSHClient
// ---------------------------------------------------------------------------

func TestPasswordCallbackAuth(t *testing.T) {
	am := PasswordCallbackAuth(func() (string, error) { return "pw", nil })
	if am == nil {
		t.Fatal("PasswordCallbackAuth returned nil")
	}
}

func TestKeyboardInteractiveAuth(t *testing.T) {
	am := KeyboardInteractiveAuth(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		return nil, nil
	})
	if am == nil {
		t.Fatal("KeyboardInteractiveAuth returned nil")
	}
}

func TestSSHClientAccessor(t *testing.T) {
	c := &Client{client: nil}
	if c.SSHClient() != nil {
		t.Error("SSHClient should return nil when client is nil")
	}
}

// ---------------------------------------------------------------------------
// AgentAuth
// ---------------------------------------------------------------------------

func TestAgentAuthNoSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := AgentAuth()
	if err == nil {
		t.Error("AgentAuth without SSH_AUTH_SOCK should fail")
	}
}

func TestAgentAuthBadSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "/nonexistent/ssh-agent.sock")
	_, err := AgentAuth()
	if err == nil {
		t.Error("AgentAuth with bad socket should fail")
	}
}

// ---------------------------------------------------------------------------
// DefaultKeyPaths
// ---------------------------------------------------------------------------

func TestDefaultKeyPaths(t *testing.T) {
	// Just verify it doesn't panic and returns a slice
	paths := DefaultKeyPaths()
	// paths may be empty or non-empty depending on the system
	_ = paths
}

// ---------------------------------------------------------------------------
// Close with jumpClient
// ---------------------------------------------------------------------------

func TestCloseWithNilJumpClient(t *testing.T) {
	// A Client whose underlying ssh.Client is nil will panic on Close
	// but jumpClient=nil path should not add errors
	c := &Client{jumpClient: nil}
	// We can't call Close() on a nil c.client, so just verify the field
	if c.jumpClient != nil {
		t.Error("jumpClient should be nil")
	}
}

// ---------------------------------------------------------------------------
// ConnectOptions struct
// ---------------------------------------------------------------------------

func TestConnectOptionsFields(t *testing.T) {
	opts := ConnectOptions{
		HostKeyAlgorithms:     "ssh-ed25519",
		PubkeyAcceptedTypes:   "ssh-ed25519",
		StrictHostKeyChecking: "no",
		UserKnownHostsFile:    "/dev/null",
	}
	if opts.HostKeyAlgorithms != "ssh-ed25519" {
		t.Error("HostKeyAlgorithms not set")
	}
	if opts.StrictHostKeyChecking != "no" {
		t.Error("StrictHostKeyChecking not set")
	}
}

// ---------------------------------------------------------------------------
// PubKeyAuth with valid ed25519 key
// ---------------------------------------------------------------------------

func TestPubKeyAuthValidKey(t *testing.T) {
	dir := t.TempDir()
	keyFile := dir + "/id_ed25519"
	key := generateTestEd25519PEM(t)
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		t.Fatal(err)
	}
	am, err := PubKeyAuth(keyFile)
	if err != nil {
		t.Fatalf("PubKeyAuth with valid key: %v", err)
	}
	if am == nil {
		t.Error("PubKeyAuth should return non-nil AuthMethod")
	}
}

func generateTestEd25519PEM(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(pemBlock)
}
