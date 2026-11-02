mock_provider "ffxf" {
  mock_resource "ffxf_database_logical_database" {
    defaults = { id = "42/analytics" }
  }
}

run "ffxf_database_logical_database_apply" {
  command = apply

  assert {
    condition     = ffxf_database_logical_database.example.id == "42/analytics"
    error_message = "The mocked ffxf_database_logical_database did not preserve its expected value."
  }
}
