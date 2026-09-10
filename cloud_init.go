package fleeting_plugin_yc

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"text/template"
)

// cloudInitExtraData is the data available to cloud_init_extra_file templates.
type cloudInitExtraData struct {
	RunnerName string
	Vars       map[string]string
}

func loadCloudInitExtraTemplate(path string) (*template.Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cloud_init_extra_file %q: %w", path, err)
	}

	tmpl, err := template.New("cloud_init_extra").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing cloud_init_extra_file %q: %w", path, err)
	}

	return tmpl, nil
}

func renderCloudInitExtra(tmpl *template.Template, runnerName string, vars map[string]string) (string, error) {
	data := cloudInitExtraData{
		RunnerName: runnerName,
		Vars:       vars,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering cloud_init_extra_file: %w", err)
	}

	return buf.String(), nil
}

// buildMultipartUserData combines #cloud-config documents into the MIME
// multipart body cloud-init expects when merging multiple user-data sources.
func buildMultipartUserData(parts ...string) (string, error) {
	var buf bytes.Buffer
	mpw := multipart.NewWriter(&buf)

	for i, content := range parts {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", `text/cloud-config; charset="us-ascii"`)
		header.Set("MIME-Version", "1.0")
		header.Set("Content-Transfer-Encoding", "7bit")
		header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="cloud-config-%d.yaml"`, i))

		pw, err := mpw.CreatePart(header)
		if err != nil {
			return "", err
		}

		if _, err := pw.Write([]byte(content)); err != nil {
			return "", err
		}
	}

	if err := mpw.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
