mock_provider "ffxf" {
  mock_resource "ffxf_policy" {
    defaults = { id = 4, normalized_document = "{}", created_at = "2026-01-01T00:00:00Z" }
  }
  mock_resource "ffxf_role" {
    defaults = { id = 6, created_at = "2026-01-01T00:00:00Z" }
  }
  mock_resource "ffxf_api_key" {
    defaults = { id = 9, role_id = 6, prefix = "ffxf_ab", enabled = true, token = "ffxf_secret", created_at = "2026-01-01T00:00:00Z" }
  }
}

run "mock_api_key" {
  command = apply
  assert {
    condition     = ffxf_api_key.ci.id == 9 && ffxf_api_key.ci.role == "readers"
    error_message = "The key should hold the role by name"
  }
  assert {
    condition     = nonsensitive(output.ci_api_key) == "ffxf_secret"
    error_message = "The secret should be exposed through the sensitive output"
  }
}
