package main

import (
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
	ycplugin "github.com/myplixa/yandex-cloud-fleeting-plugin"
)

func main() {
	plugin.Serve(&ycplugin.InstanceGroup{})
}
