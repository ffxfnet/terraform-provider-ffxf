mock_provider "ffxf" {
  mock_data "ffxf_databases" {
    defaults = { quota = 5, databases_json = "[]" }
  }
}

run "ffxf_databases_plan" {
  command = plan

  assert {
    condition     = data.ffxf_databases.all.quota == 5
    error_message = "The mocked ffxf_databases did not return its expected value."
  }
}
