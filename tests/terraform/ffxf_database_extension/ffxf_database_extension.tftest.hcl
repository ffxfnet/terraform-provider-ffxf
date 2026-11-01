mock_provider "ffxf" {
  mock_resource "ffxf_database_extension" {
    defaults = { id = "42/defaultdb/vector" }
  }
}

run "ffxf_database_extension_apply" {
  command = apply

  assert {
    condition     = ffxf_database_extension.example.id == "42/defaultdb/vector"
    error_message = "The mocked ffxf_database_extension did not preserve its expected value."
  }
}
