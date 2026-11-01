package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDatabaseAreaConstructors(t *testing.T) {
	resourceNames := map[string]bool{}
	for _, constructor := range databaseAreaResources() {
		var metadata resource.MetadataResponse
		constructor().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		resourceNames[metadata.TypeName] = true
	}
	if len(resourceNames) != 9 {
		t.Fatalf("database resource types = %d, want 9", len(resourceNames))
	}
	dataSourceNames := map[string]bool{}
	for _, constructor := range databaseAreaDataSources() {
		var metadata datasource.MetadataResponse
		constructor().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		dataSourceNames[metadata.TypeName] = true
	}
	if len(dataSourceNames) != 6 {
		t.Fatalf("database data source types = %d, want 6", len(dataSourceNames))
	}
}
