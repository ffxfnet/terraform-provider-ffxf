mock_provider "ffxf" {
  mock_data "ffxf_database_backups" {
    defaults = { backups_json = "[]", restore_window_json = "{}" }
  }
}

run "ffxf_database_backups_plan" {
  command = plan

  assert {
    condition     = data.ffxf_database_backups.current.backups_json == "[]"
    error_message = "The mocked ffxf_database_backups did not return its expected value."
  }
}
