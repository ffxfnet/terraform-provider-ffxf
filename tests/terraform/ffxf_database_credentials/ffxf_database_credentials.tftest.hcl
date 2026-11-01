mock_provider "ffxf" {
  mock_data "ffxf_database_credentials" {
    defaults = { credentials_json = "{}", users_json = "[]", connection_uri = "postgres://mock", public_connection_uri = "" }
  }
}

run "ffxf_database_credentials_plan" {
  command = plan

  assert {
    condition     = data.ffxf_database_credentials.current.connection_uri == "postgres://mock"
    error_message = "The mocked ffxf_database_credentials did not return its expected value."
  }
}
