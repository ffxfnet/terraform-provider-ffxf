mock_provider "ffxf" {
  mock_resource "ffxf_database_fork" {
    defaults = { id = 43, status = "active", engine = "postgresql", version = "18" }
  }
}

run "ffxf_database_fork_apply" {
  command = apply

  assert {
    condition     = ffxf_database_fork.example.id == 43
    error_message = "The mocked ffxf_database_fork did not preserve its expected value."
  }
}
