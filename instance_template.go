package fleeting_plugin_yc

type InstanceTemplate struct {
	Platform         string `yaml:"platform"`
	Preemptible      bool   `yaml:"preemptible"`
	Zone             string `yaml:"zone"`
	ServiceAccount   string `yaml:"serviceAccount"`
	VmServiceAccount string `yaml:"vmServiceAccount,omitempty"`
	Resources        struct {
		CPU      int64 `yaml:"cpu,omitempty"`
		Memory   int64 `yaml:"memory,omitempty"`
		Fraction int64 `yaml:"fraction,omitempty"`
	} `yaml:"resources"`
	Disk struct {
		Type  string `yaml:"type,omitempty"`
		Size  int64  `yaml:"size,omitempty"`
		Image string `yaml:"image,omitempty"`
	} `yaml:"disk"`
	Network struct {
		Network  string `yaml:"network,omitempty"`
		Subnet   string `yaml:"subnet,omitempty"`
		Nat      bool   `yaml:"nat,omitempty"`
		Security string `yaml:"security,omitempty"`
	} `yaml:"network"`
	Monitoring struct {
		Enabled bool `yaml:"enabled,omitempty"`
	} `yaml:"monitoring, omitempty"`

	// CloudInitExtraFile is an optional path to a #cloud-config document
	// (parsed as a Go text/template, see cloud_init.go) merged alongside the
	// plugin's own generated user-data (e.g. the monitoring-agent installer)
	// via real cloud-init multipart MIME - lets operators install Docker,
	// CA certs, etc. without needing to bake a separate custom image per
	// tweak.
	CloudInitExtraFile string `yaml:"cloud_init_extra_file,omitempty"`
	// CloudInitVars are made available to the CloudInitExtraFile template as
	// {{ .Vars.<key> }} - the place for anything the template needs that
	// shouldn't be hardcoded into the file itself (e.g. a registry token).
	CloudInitVars map[string]string `yaml:"cloud_init_vars,omitempty"`
}
