package fleeting_plugin_yc

import (
	"testing"

	"github.com/ghodss/yaml"
)

// TestInstanceTemplate_UnmarshalsSnakeCaseFields guards against a bug where
// CloudInitExtraFile/CloudInitVars had only `yaml:` struct tags while the
// plugin actually parses template.yml via github.com/ghodss/yaml, which
// converts to JSON and unmarshals with encoding/json - that package ignores
// `yaml:` tags entirely and falls back to case-insensitive field-name
// matching, which silently drops any snake_case key. Every other field in
// InstanceTemplate happens to be single-word/camelCase and survived that
// fallback by coincidence, masking the bug.
func TestInstanceTemplate_UnmarshalsSnakeCaseFields(t *testing.T) {
	raw := []byte(`
platform: "standard-v3"
zone: "ru-central1-a"
preemptible: true
serviceAccount: "sa1"
vmServiceAccount: "sa2"
disk:
  type: "network-ssd"
  size: 17179869184
  image: "fd8image"
network:
  subnet: "subnet1"
cloud_init_extra_file: "/etc/gitlab-runner/cloud-init-extra.yaml"
cloud_init_vars:
  username: "ci"
`)

	var tpl InstanceTemplate
	if err := yaml.Unmarshal(raw, &tpl); err != nil {
		t.Fatalf("unmarshalling template: %v", err)
	}

	if tpl.CloudInitExtraFile != "/etc/gitlab-runner/cloud-init-extra.yaml" {
		t.Fatalf("cloud_init_extra_file did not parse, got %q", tpl.CloudInitExtraFile)
	}
	if tpl.CloudInitVars["username"] != "ci" {
		t.Fatalf("cloud_init_vars.username did not parse, got %q", tpl.CloudInitVars["username"])
	}
}
