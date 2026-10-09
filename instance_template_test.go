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

// TestInstanceGroup_ZoneIDs_SubnetIDs_FallBackToLegacySingularFields guards
// the multi-zone upgrade path: existing template.yml files only set the
// legacy singular `zone`/`network.subnet` fields, and must keep working
// unchanged - zoneIDs/subnetIDs should only switch to the plural fields when
// the config actually sets them.
func TestInstanceGroup_ZoneIDs_SubnetIDs_FallBackToLegacySingularFields(t *testing.T) {
	g := &InstanceGroup{template: InstanceTemplate{Zone: "ru-central1-a"}}
	g.template.Network.Subnet = "subnet1"

	if got := g.zoneIDs(); len(got) != 1 || got[0] != "ru-central1-a" {
		t.Fatalf("zoneIDs() with no Zones set = %v, want [ru-central1-a]", got)
	}
	if got := g.subnetIDs(); len(got) != 1 || got[0] != "subnet1" {
		t.Fatalf("subnetIDs() with no Subnets set = %v, want [subnet1]", got)
	}
}

func TestInstanceGroup_ZoneIDs_SubnetIDs_PreferPluralFieldsWhenSet(t *testing.T) {
	g := &InstanceGroup{template: InstanceTemplate{
		Zone:  "ru-central1-a",
		Zones: []string{"ru-central1-a", "ru-central1-b"},
	}}
	g.template.Network.Subnet = "subnet1"
	g.template.Network.Subnets = []string{"subnet1", "subnet2"}

	wantZones := []string{"ru-central1-a", "ru-central1-b"}
	if got := g.zoneIDs(); !equalStrings(got, wantZones) {
		t.Fatalf("zoneIDs() = %v, want %v", got, wantZones)
	}

	wantSubnets := []string{"subnet1", "subnet2"}
	if got := g.subnetIDs(); !equalStrings(got, wantSubnets) {
		t.Fatalf("subnetIDs() = %v, want %v", got, wantSubnets)
	}
}

func TestInstanceTemplate_UnmarshalsMultiZoneFields(t *testing.T) {
	raw := []byte(`
platform: "standard-v3"
zones: ["ru-central1-a", "ru-central1-b"]
preemptible: true
serviceAccount: "sa1"
network:
  subnets: ["subnet1", "subnet2"]
`)

	var tpl InstanceTemplate
	if err := yaml.Unmarshal(raw, &tpl); err != nil {
		t.Fatalf("unmarshalling template: %v", err)
	}

	wantZones := []string{"ru-central1-a", "ru-central1-b"}
	if !equalStrings(tpl.Zones, wantZones) {
		t.Fatalf("zones did not parse, got %v", tpl.Zones)
	}

	wantSubnets := []string{"subnet1", "subnet2"}
	if !equalStrings(tpl.Network.Subnets, wantSubnets) {
		t.Fatalf("network.subnets did not parse, got %v", tpl.Network.Subnets)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
