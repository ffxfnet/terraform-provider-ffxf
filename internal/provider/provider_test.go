package provider

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderMetadata(t *testing.T) {
	provider := &FFXFProvider{version: "1.2.3"}
	var response frameworkprovider.MetadataResponse

	provider.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &response)

	if response.TypeName != "ffxf" {
		t.Errorf("Metadata() TypeName = %q, want %q", response.TypeName, "ffxf")
	}
	if response.Version != "1.2.3" {
		t.Errorf("Metadata() Version = %q, want %q", response.Version, "1.2.3")
	}
}

func TestProviderSchema(t *testing.T) {
	provider := &FFXFProvider{}
	var response frameworkprovider.SchemaResponse

	provider.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &response)

	for _, name := range []string{"endpoint", "token"} {
		if _, ok := response.Schema.Attributes[name]; !ok {
			t.Errorf("Schema() is missing the %q attribute", name)
		}
	}
}

func TestInstanceResourceMetadata(t *testing.T) {
	instance := NewInstanceResource()
	var response resource.MetadataResponse

	instance.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &response)

	if response.TypeName != "ffxf_instance" {
		t.Errorf("Metadata() TypeName = %q, want %q", response.TypeName, "ffxf_instance")
	}
}
