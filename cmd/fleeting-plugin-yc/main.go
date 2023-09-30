package main

import (
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
	ycplugin "gitlab.com/k.mukhin/fleeting-plugin-yc"
)

func main() {
	plugin.Serve(&ycplugin.InstanceGroup{})
}
