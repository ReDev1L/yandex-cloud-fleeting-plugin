package fleeting_plugin_yc

import (
	"os"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func minimalTemplate(preemptible bool) InstanceTemplate {
	var tpl InstanceTemplate
	tpl.Platform = "standard-v3"
	tpl.Zone = "ru-central1-a"
	tpl.Preemptible = preemptible
	tpl.Resources.CPU = 2
	tpl.Resources.Memory = 2147483648
	tpl.Disk.Type = "network-ssd"
	tpl.Disk.Size = 17179869184
	tpl.Disk.Image = "fd8image"
	tpl.Network.Network = "net1"
	tpl.Network.Subnet = "subnet1"
	return tpl
}

func newTestGroup(preemptible bool) *InstanceGroup {
	g := &InstanceGroup{Name: "test-runners"}
	g.template = minimalTemplate(preemptible)
	g.settings = provider.Settings{
		ConnectorConfig: provider.ConnectorConfig{Username: "ci"},
	}
	g.SshPublicKey = "ssh-rsa AAAAtest"
	return g
}

// TestPreemptible_HonorsTemplateField guards against the upstream bug where
// SchedulingPolicy.Preemptible was hardcoded to true regardless of what the
// operator set in template.yml's preemptible field.
func TestPreemptible_HonorsTemplateField(t *testing.T) {
	for _, want := range []bool{true, false} {
		g := newTestGroup(want)
		tpl := g.GenerateInstanceGroup()
		if got := tpl.SchedulingPolicy.Preemptible; got != want {
			t.Fatalf("preemptible=%v in template.yml, but SchedulingPolicy.Preemptible=%v", want, got)
		}
	}
}

// TestGenerateMeta_NoExtras verifies the plain case (no monitoring, no
// cloud_init_extra_file) still sets ssh-keys and nothing else weird.
func TestGenerateMeta_NoExtras(t *testing.T) {
	g := newTestGroup(false)
	meta := g.GenerateMeta()

	if meta["ssh-keys"] != "ci:ssh-rsa AAAAtest" {
		t.Fatalf("unexpected ssh-keys value: %q", meta["ssh-keys"])
	}
	if _, ok := meta["user-data"]; ok {
		t.Fatalf("expected no user-data when monitoring is disabled and no cloud_init_extra_file is set, got: %q", meta["user-data"])
	}
}

// TestGenerateMeta_MonitoringOnly verifies the pre-existing monitoring-agent
// behavior is unchanged by the GenerateMeta rewrite.
func TestGenerateMeta_MonitoringOnly(t *testing.T) {
	g := newTestGroup(false)
	g.template.Monitoring.Enabled = true

	meta := g.GenerateMeta()

	if meta["install-unified-agent"] != "1" {
		t.Fatalf("expected install-unified-agent=1, got %q", meta["install-unified-agent"])
	}
	if meta["user-data"] == "" || !strings.Contains(meta["user-data"], "unifiedAgent") {
		t.Fatalf("expected user-data to contain the unified agent installer, got: %q", meta["user-data"])
	}
}

// TestGenerateMeta_MonitoringPlusExtra_MergesAsMultipart verifies that when
// both monitoring and a cloud_init_extra_file are set, both end up present
// (via multipart) instead of one clobbering the other - this is the actual
// gap this fork closes relative to upstream, which had no extensibility
// mechanism at all.
func TestGenerateMeta_MonitoringPlusExtra_MergesAsMultipart(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/extra.yaml"
	if err := os.WriteFile(path, []byte("#cloud-config\npackages:\n  - docker.io\n"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	tmpl, err := loadCloudInitExtraTemplate(path)
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}

	g := newTestGroup(false)
	g.template.Monitoring.Enabled = true
	g.cloudInitExtraTemplate = tmpl

	meta := g.GenerateMeta()

	userData := meta["user-data"]
	if !strings.Contains(userData, "unifiedAgent") {
		t.Fatalf("expected monitoring part to still be present, got: %q", userData)
	}
	if !strings.Contains(userData, "docker.io") {
		t.Fatalf("expected cloud_init_extra_file content to be present, got: %q", userData)
	}
	if !strings.Contains(userData, "boundary") && !strings.Contains(userData, "--") {
		t.Fatalf("expected multipart MIME framing when merging 2 parts, got: %q", userData)
	}
}
