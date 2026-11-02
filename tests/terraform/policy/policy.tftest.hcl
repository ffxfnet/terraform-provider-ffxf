mock_provider "ffxf" {
  mock_resource "ffxf_policy" {
    defaults = { id = 4, normalized_document = "{}", created_at = "2026-01-01T00:00:00Z" }
  }
}

run "mock_policy" {
  command = apply
  assert {
    condition     = ffxf_policy.readers.id == 4 && ffxf_policy.readers.name == "vm-readers"
    error_message = "Policy ID should come from the mock"
  }
  assert {
    condition     = jsondecode(ffxf_policy.readers.document).statements[0].actions[0] == "vm:read"
    error_message = "The document should be the encoded configuration"
  }
}
