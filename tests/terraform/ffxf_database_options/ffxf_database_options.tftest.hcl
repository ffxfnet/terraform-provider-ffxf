mock_provider "ffxf" {
  mock_data "ffxf_database_options" {
    defaults = { options_json = "{}" }
  }
}

run "ffxf_database_options_plan" {
  command = plan

  assert {
    condition     = data.ffxf_database_options.available.options_json == "{}"
    error_message = "The mocked ffxf_database_options did not return its expected value."
  }
}
