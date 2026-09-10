package main

import (
	ycplugin "github.com/myplixa/yandex-cloud-fleeting-plugin"
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

func main() {
	plugin.Serve(&ycplugin.InstanceGroup{})
}
