mock_provider "ffxf" {
  mock_data "ffxf_database_queries" {
    defaults = { queries_json = "[]" }
  }
}

run "ffxf_database_queries_plan" {
  command = plan

  assert {
    condition     = data.ffxf_database_queries.current.queries_json == "[]"
    error_message = "The mocked ffxf_database_queries did not return its expected value."
  }
}
