package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"terraform-provider-IncidentRelay/incidentrelay"
)

func main() {
	var debug bool
	var address string

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with debugger support")
	flag.StringVar(&address, "address", "registry.terraform.io/incidentrelay/incidentrelay", "provider address used for debugging")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: address,
		ProviderFunc: incidentrelay.Provider,
	})
}
