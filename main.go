package main

import (
	"context"
	"flag"
	"log"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Active le mode debug du provider")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.opentofu.org/ffxfnet/ffxf",
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
