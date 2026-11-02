mock_provider "ffxf" {
  mock_resource "ffxf_policy" {
    defaults = { id = 4, normalized_document = "{}", created_at = "2026-01-01T00:00:00Z" }
  }
  mock_resource "ffxf_role" {
    defaults = { id = 6, created_at = "2026-01-01T00:00:00Z" }
  }
  mock_resource "ffxf_oidc_binding" {
    defaults = { id = 3, role_id = 6, enabled = true, created_at = "2026-01-01T00:00:00Z" }
  }
  mock_data "ffxf_oidc_providers" {
    defaults = { providers = [{ id = 2, name = "GitHub", issuer = "https://token.actions.githubusercontent.com", audience = "ffxf", enabled = true }] }
  }
}

run "mock_oidc_binding" {
  command = apply
  assert {
    condition     = ffxf_oidc_binding.github.provider_id == 2 && ffxf_oidc_binding.github.id == 3
    error_message = "The binding should use the provider found by the data source"
  }
}
