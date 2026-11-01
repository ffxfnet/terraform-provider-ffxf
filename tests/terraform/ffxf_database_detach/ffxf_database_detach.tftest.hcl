mock_provider "ffxf" {
  mock_resource "ffxf_database_detach" {
    defaults = { id = "43" }
  }
}

run "ffxf_database_detach_apply" {
  command = apply

  assert {
    condition     = ffxf_database_detach.example.id == "43"
    error_message = "The mocked ffxf_database_detach did not preserve its expected value."
  }
}
