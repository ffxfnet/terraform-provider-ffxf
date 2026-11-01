mock_provider "ffxf" {
  mock_resource "ffxf_database_restore" {
    defaults = { id = 44, status = "active", engine = "postgresql", version = "18" }
  }
}

run "ffxf_database_restore_apply" {
  command = apply

  assert {
    condition     = ffxf_database_restore.example.id == 44
    error_message = "The mocked ffxf_database_restore did not preserve its expected value."
  }
}
