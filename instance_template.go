package fleeting_plugin_yc

// Field tags are `json:`, not `yaml:` - see README.md#instance-template for why.
type InstanceTemplate struct {
	Platform         string `json:"platform"`
	Preemptible      bool   `json:"preemptible"`
	Zone             string `json:"zone"`
	ServiceAccount   string `json:"serviceAccount"`
	VmServiceAccount string `json:"vmServiceAccount,omitempty"`
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
		Network  string `json:"network,omitempty"`
		Subnet   string `json:"subnet,omitempty"`
		Nat      bool   `json:"nat,omitempty"`
		Security string `json:"security,omitempty"`
	} `json:"network"`
	Monitoring struct {
		Enabled bool `json:"enabled,omitempty"`
	} `json:"monitoring,omitempty"`

	// See README.md#extending-cloud-init for both fields.
	CloudInitExtraFile string            `json:"cloud_init_extra_file,omitempty"`
	CloudInitVars      map[string]string `json:"cloud_init_vars,omitempty"`
}
