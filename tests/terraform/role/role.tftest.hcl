mock_provider "ffxf" {
  mock_resource "ffxf_policy" {
    defaults = { id = 4, normalized_document = "{}", created_at = "2026-01-01T00:00:00Z" }
  }
  mock_resource "ffxf_role" {
    defaults = { id = 6, created_at = "2026-01-01T00:00:00Z" }
  }
}

run "mock_role" {
  command = apply
  assert {
    condition     = ffxf_role.readers.id == 6 && contains(ffxf_role.readers.policy_ids, 4)
    error_message = "The role should carry the policy created in the same configuration"
  }
}
