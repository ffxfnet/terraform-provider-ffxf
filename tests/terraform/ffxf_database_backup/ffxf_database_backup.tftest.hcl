mock_provider "ffxf" {
  mock_resource "ffxf_database_backup" {
    defaults = { id = "42" }
  }
}

run "ffxf_database_backup_apply" {
  command = apply

  assert {
    condition     = ffxf_database_backup.example.id == "42"
    error_message = "The mocked ffxf_database_backup did not preserve its expected value."
  }
}
