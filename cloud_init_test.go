package fleeting_plugin_yc

import (
	"mime"
	"mime/multipart"
	"os"
	"strings"
	"testing"
	"text/template"
)

func TestLoadCloudInitExtraTemplate_MissingFile(t *testing.T) {
	if _, err := loadCloudInitExtraTemplate("/nonexistent/path/does-not-exist.yaml"); err == nil {
		t.Fatal("expected an error for a missing cloud_init_extra_file, got nil")
	}
}

func TestLoadCloudInitExtraTemplate_InvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.yaml"
	if err := os.WriteFile(path, []byte("{{ .Unclosed"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if _, err := loadCloudInitExtraTemplate(path); err == nil {
		t.Fatal("expected a parse error for a malformed template, got nil")
	}
}

func TestRenderCloudInitExtra(t *testing.T) {
	tmpl, err := template.New("t").Parse("runner={{ .RunnerName }} token={{ .Vars.token }}")
	if err != nil {
		t.Fatalf("parsing template: %v", err)
	}

	out, err := renderCloudInitExtra(tmpl, "my-runner", map[string]string{"token": "secret123"})
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	want := "runner=my-runner token=secret123"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestBuildMultipartUserData_ProducesParsableMultipart(t *testing.T) {
	partA := "#cloud-config\nruncmd:\n  - echo a\n"
	partB := "#cloud-config\nruncmd:\n  - echo b\n"

	combined, err := buildMultipartUserData(partA, partB)
	if err != nil {
		t.Fatalf("buildMultipartUserData: %v", err)
	}

	if !strings.HasPrefix(combined, "--") {
		t.Fatalf("expected output to start with a MIME boundary, got: %q", combined[:min(40, len(combined))])
	}

	boundary, err := extractBoundary(combined)
	if err != nil {
		t.Fatalf("extracting boundary: %v", err)
	}

	mr := multipart.NewReader(strings.NewReader(combined), boundary)
	var parts []string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		buf := make([]byte, 4096)
		n, _ := p.Read(buf)
		parts = append(parts, string(buf[:n]))
	}

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %v", len(parts), parts)
	}
	if parts[0] != partA || parts[1] != partB {
		t.Fatalf("parts did not round-trip correctly: %q / %q", parts[0], parts[1])
	}
}

func TestBuildMultipartUserData_SinglePartStillValid(t *testing.T) {
	// GenerateMeta only calls buildMultipartUserData when there are 2+
	// parts, but the function itself should still behave sanely with one.
	combined, err := buildMultipartUserData("#cloud-config\nfinal_message: done\n")
	if err != nil {
		t.Fatalf("buildMultipartUserData: %v", err)
	}
	if !strings.Contains(combined, "final_message: done") {
		t.Fatalf("expected content to be present in output, got: %q", combined)
	}
}

// --- helpers ---

func extractBoundary(multipartBody string) (string, error) {
	firstLine := strings.SplitN(multipartBody, "\r\n", 2)[0]
	if !strings.HasPrefix(firstLine, "--") {
		return "", mime.ErrInvalidMediaParameter
	}
	return strings.TrimPrefix(firstLine, "--"), nil
}
