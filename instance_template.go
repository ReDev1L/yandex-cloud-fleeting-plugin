package fleeting_plugin_yc

// Field tags are `json:`, not `yaml:` - see README.md#instance-template for why.
type InstanceTemplate struct {
	Platform    string `json:"platform"`
	Preemptible bool   `json:"preemptible"`
	// Zone is the legacy single-zone field. Zones, if non-empty, takes
	// priority over it - see InstanceGroup.zoneIDs.
	Zone             string   `json:"zone,omitempty"`
	Zones            []string `json:"zones,omitempty"`
	ServiceAccount   string   `json:"serviceAccount"`
	VmServiceAccount string   `json:"vmServiceAccount,omitempty"`
	Resources        struct {
		CPU      int64 `json:"cpu,omitempty"`
		Memory   int64 `json:"memory,omitempty"`
		Fraction int64 `json:"fraction,omitempty"`
	} `json:"resources"`
	Disk struct {
		Type  string `json:"type,omitempty"`
		Size  int64  `json:"size,omitempty"`
		Image string `json:"image,omitempty"`
	} `json:"disk"`
	Network struct {
		Network string `json:"network,omitempty"`
		// Subnet is the legacy single-subnet field. Subnets, if non-empty,
		// takes priority over it - see InstanceGroup.subnetIDs. Each entry
		// must belong to the matching zone in Zones (YC pairs subnet to
		// zone by each subnet's own zone, not by list position).
		Subnet   string   `json:"subnet,omitempty"`
		Subnets  []string `json:"subnets,omitempty"`
		Nat      bool     `json:"nat,omitempty"`
		Security string   `json:"security,omitempty"`
	} `json:"network"`
	Monitoring struct {
		Enabled bool `json:"enabled,omitempty"`
	} `json:"monitoring,omitempty"`

	// See README.md#extending-cloud-init for both fields.
	CloudInitExtraFile string            `json:"cloud_init_extra_file,omitempty"`
	CloudInitVars      map[string]string `json:"cloud_init_vars,omitempty"`
}
