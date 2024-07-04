package main

import (
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/plugin"
	"os"
	"packer-plugin-ecloud/builder/ecloud/basic"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("basic", new(basic.Builder))
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
