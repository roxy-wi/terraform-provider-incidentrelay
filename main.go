package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/roxy-wi/terraform-provider-incidentrelay/incidentrelay"
)

func main() {
	var debug bool
	var address string

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with debugger support")
	flag.StringVar(&address, "address", "registry.terraform.io/roxy-wi/incidentrelay", "provider address used for debugging")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: address,
		ProviderFunc: incidentrelay.Provider,
	})
}
