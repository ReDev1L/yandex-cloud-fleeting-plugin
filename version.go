package fleeting_plugin_yc

import (
	"fmt"
	"runtime"
)

var (
	NAME      = "fleeting-plugin-yc"
	VERSION   = "dev"
	REVISION  = "HEAD"
	REFERENCE = "HEAD"
	BUILT     = "now"
)

func BuildInfo() string {
	return fmt.Sprintf("sha=%s; ref=%s; go=%s; built_at=%s; os_arch=%s/%s",
		REVISION, REFERENCE, runtime.Version(), BUILT, runtime.GOOS, runtime.GOARCH)
}
