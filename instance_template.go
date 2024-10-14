package fleeting_plugin_yc

type InstanceTemplate struct {
	Platform       string `yaml:"platform"`
	Preemptible    bool   `yaml:"preemptible"`
	Zone           string `yaml:"zone"`
	ServiceAccount string `yaml:"serviceAccount"`
	Resources      struct {
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
}
