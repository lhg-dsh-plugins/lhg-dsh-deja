package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hyphenatedProjectStore builds a store holding one session recorded in a
// project directory with a hyphen in its name — proj/my-app — and returns
// that directory. The rows name such a project proj/my-app; the encoded-dir
// name read back the lossy way would call it my/app (#5321).
func hyphenatedProjectStore(t *testing.T) string {
	t.Helper()
	tmp := hermeticEnv(t)
	t.Setenv("DEJA_INDEX_DIR", filepath.Join(tmp, "index.db"))
	proj := filepath.Join(tmp, "proj", "my-app")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	// `files` only lists recorded files under a repository on this disk, so
	// the project has to be one.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if out, err := exec.Command("git", "-C", proj, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(proj, "notes.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("DEJA_CLAUDE_ROOT")
	enc := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, proj)
	writeClaudeFixture(t, filepath.Join(root, enc, "s1.jsonl"), "s1", []string{
		`{"type":"user","sessionId":"s1","cwd":"` + proj + `","timestamp":"2026-07-01T10:00:00Z","message":{"role":"user","content":"have a subagent write the release notes into notes.txt"}}`,
		`{"type":"assistant","sessionId":"s1","cwd":"` + proj + `","timestamp":"2026-07-01T10:01:00Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"make release"}}]}}`,
		`{"type":"assistant","sessionId":"s1","cwd":"` + proj + `","timestamp":"2026-07-01T10:02:00Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"` + filepath.Join(proj, "notes.txt") + `","old_string":"a","new_string":"b"}}]}}`,
	})
	if _, err := captureRun(t, "index"); err != nil {
		t.Fatal(err)
	}
	return proj
}

// The scope line under `files` named the first candidate — Claude's lossy
// encoded-dir name — so a hyphenated project read as a name no row shows and
// --project does not accept (#5321).
func TestFilesFooterNamesAHyphenatedProjectTheWayTheRowsDo(t *testing.T) {
	proj := hyphenatedProjectStore(t)
	t.Chdir(proj)
	out, err := captureRun(t, "files", "release")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "files touched while working on") {
		t.Fatalf("the fixture no longer answers, so this pins nothing:\n%s", out)
	}
	if !strings.Contains(out, "— in proj/my-app only") {
		t.Fatalf("footer does not name the project the rows do:\n%s", out)
	}
	if strings.Contains(out, "in my/app only") {
		t.Fatalf("footer names the lossy encoded-dir form:\n%s", out)
	}
}

// The same line under `how` took the same name from the same helper.
func TestHowFooterNamesAHyphenatedProjectTheWayTheRowsDo(t *testing.T) {
	proj := hyphenatedProjectStore(t)
	t.Chdir(proj)
	out, err := captureRun(t, "how", "release")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "make release") {
		t.Fatalf("the fixture no longer answers, so this pins nothing:\n%s", out)
	}
	if !strings.Contains(out, "— in proj/my-app only") {
		t.Fatalf("footer does not name the project the rows do:\n%s", out)
	}
}

// And the name --json carries is the same one.
func TestFilesJSONNamesAHyphenatedProjectTheWayTheRowsDo(t *testing.T) {
	proj := hyphenatedProjectStore(t)
	t.Chdir(proj)
	out, err := captureRun(t, "files", "release", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var env filesJSON
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("files --json did not parse: %v\n%s", err, out)
	}
	if env.Project != "proj/my-app" {
		t.Fatalf("project = %q, want the name the rows show, proj/my-app", env.Project)
	}
}
