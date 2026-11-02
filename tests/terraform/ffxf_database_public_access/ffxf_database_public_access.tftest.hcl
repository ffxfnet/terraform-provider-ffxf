mock_provider "ffxf" {
  mock_resource "ffxf_database_public_access" {
    defaults = { id = "42" }
  }
}

run "ffxf_database_public_access_apply" {
  command = apply

  assert {
    condition     = ffxf_database_public_access.example.id == "42"
    error_message = "The mocked ffxf_database_public_access did not preserve its expected value."
  }
}
