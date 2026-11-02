package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDatabaseConfigValidatorRegistration(t *testing.T) {
	if got := (&databaseResource{}).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("database validators = %d, want 1", len(got))
	}
	if got := (&databaseDerivedResource{kind: "restore"}).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("restore validators = %d, want 1", len(got))
	}
	if got := (&databaseDerivedResource{kind: "fork"}).ConfigValidators(context.Background()); len(got) != 0 {
		t.Fatalf("fork validators = %d, want 0", len(got))
	}
	if got := (&databaseDataSource{kind: "_metrics"}).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("metrics validators = %d, want 1", len(got))
	}
	if got := (&databaseDataSource{kind: "_queries"}).ConfigValidators(context.Background()); len(got) != 0 {
		t.Fatalf("queries validators = %d, want 0", len(got))
	}
	var _ resource.ConfigValidator = databaseConfigValidator{}
	var _ datasource.ConfigValidator = databaseConfigValidator{}
}
