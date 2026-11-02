mock_provider "ffxf" {
  mock_resource "ffxf_ssh_key" {
    defaults = { id = 12, fingerprint = "SHA256:example", created_at = "2026-01-01T00:00:00Z" }
  }
}

run "mock_ssh_key" {
  command = apply
  assert {
    condition     = ffxf_ssh_key.work.id == 12
    error_message = "SSH key ID should come from the mock"
  }
}
