package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_FindsSkillFiles(t *testing.T) {
	dir := t.TempDir()

	// Create two skills in separate subdirectories.
	mkSkill(t, dir, "alpha", "---\nname: alpha\ndescription: Alpha skill\n---\n# Alpha")
	mkSkill(t, dir, "beta", "---\nname: beta\ndescription: Beta skill\n---\n# Beta")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if got := len(reg.Skills()); got != 2 {
		t.Fatalf("expected 2 skills, got %d", got)
	}

	names := map[string]bool{}
	for _, sk := range reg.Skills() {
		names[sk.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("expected alpha and beta, got %v", names)
	}
}

func TestScan_ParsesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	mkSkill(t, dir, "myskill", "---\nname: myskill\ndescription: Does useful things\n---\nBody content here")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	skills := reg.Skills()
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	sk := skills[0]
	if sk.Name != "myskill" {
		t.Errorf("name = %q, want %q", sk.Name, "myskill")
	}
	if sk.Description != "Does useful things" {
		t.Errorf("description = %q, want %q", sk.Description, "Does useful things")
	}
	if sk.Path == "" {
		t.Error("path should not be empty")
	}
}

func TestScan_MultipleDirs(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	mkSkill(t, dirA, "a", "---\nname: a\ndescription: from dir A\n---\n")
	mkSkill(t, dirB, "b", "---\nname: b\ndescription: from dir B\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dirA, dirB}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if got := len(reg.Skills()); got != 2 {
		t.Fatalf("expected 2 skills, got %d", got)
	}
}

func TestScan_SkipsMalformedFiles(t *testing.T) {
	dir := t.TempDir()

	// Valid skill.
	mkSkill(t, dir, "good", "---\nname: good\ndescription: Valid\n---\n")

	// Missing opening delimiter.
	mkSkill(t, dir, "no-delim", "name: bad\ndescription: No delimiter\n")

	// Empty frontmatter.
	mkSkill(t, dir, "empty-fm", "---\n---\n")

	// Missing name field.
	mkSkill(t, dir, "no-name", "---\ndescription: No name field\n---\n")

	// Invalid YAML.
	mkSkill(t, dir, "bad-yaml", "---\n: :\n  [invalid\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	skills := reg.Skills()
	if len(skills) != 1 {
		t.Fatalf("expected 1 valid skill, got %d", len(skills))
	}
	if skills[0].Name != "good" {
		t.Errorf("expected 'good', got %q", skills[0].Name)
	}
}

func TestScan_MissingDirectory(t *testing.T) {
	var reg Registry
	// A missing directory is handled gracefully (no skills found, no error).
	err := reg.Scan([]string{"/nonexistent/path/that/does/not/exist"})
	if err != nil {
		t.Fatalf("expected graceful handling, got error: %v", err)
	}
	if got := len(reg.Skills()); got != 0 {
		t.Fatalf("expected 0 skills, got %d", got)
	}
}

func TestScan_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if got := len(reg.Skills()); got != 0 {
		t.Fatalf("expected 0 skills, got %d", got)
	}
}

func TestScan_ResetsOnRescan(t *testing.T) {
	dir := t.TempDir()
	mkSkill(t, dir, "first", "---\nname: first\ndescription: First\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got := len(reg.Skills()); got != 1 {
		t.Fatalf("expected 1 skill, got %d", got)
	}

	// Scan again with empty dir — should reset.
	emptyDir := t.TempDir()
	if err := reg.Scan([]string{emptyDir}); err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if got := len(reg.Skills()); got != 0 {
		t.Fatalf("expected 0 skills after rescan, got %d", got)
	}
}

func TestFilter_AllowlistFilters(t *testing.T) {
	dir := t.TempDir()
	mkSkill(t, dir, "a", "---\nname: a\ndescription: A\n---\n")
	mkSkill(t, dir, "b", "---\nname: b\ndescription: B\n---\n")
	mkSkill(t, dir, "c", "---\nname: c\ndescription: C\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	filtered := reg.Filter([]string{"a", "c"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered skills, got %d", len(filtered))
	}

	names := map[string]bool{}
	for _, sk := range filtered {
		names[sk.Name] = true
	}
	if !names["a"] || !names["c"] {
		t.Errorf("expected a and c, got %v", names)
	}
	if names["b"] {
		t.Error("b should have been filtered out")
	}
}

func TestFilter_EmptyAllowlistReturnsAll(t *testing.T) {
	dir := t.TempDir()
	mkSkill(t, dir, "x", "---\nname: x\ndescription: X\n---\n")
	mkSkill(t, dir, "y", "---\nname: y\ndescription: Y\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	filtered := reg.Filter(nil)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 skills with nil allowlist, got %d", len(filtered))
	}

	filtered = reg.Filter([]string{})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 skills with empty allowlist, got %d", len(filtered))
	}
}

func TestFilter_NoMatchReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	mkSkill(t, dir, "x", "---\nname: x\ndescription: X\n---\n")

	var reg Registry
	if err := reg.Scan([]string{dir}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	filtered := reg.Filter([]string{"nonexistent"})
	if len(filtered) != 0 {
		t.Fatalf("expected 0 filtered skills, got %d", len(filtered))
	}
}

// mkSkill creates a subdirectory with a SKILL.md file.
func mkSkill(t *testing.T, parent, name, content string) {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
