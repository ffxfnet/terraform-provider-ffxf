mock_provider "ffxf" {
  mock_resource "ffxf_database" {
    defaults = { id = 42, status = "active", engine_version = "18.1", region = "montreal", connection_json = "{}", public_access_json = "{}", disk_json = "{}", users_json = "[]", logical_databases_json = "[]", created_at = "2026-01-01T00:00:00Z" }
  }
}

run "ffxf_database_apply" {
  command = apply

  assert {
    condition     = ffxf_database.example.id == 42
    error_message = "The mocked ffxf_database did not preserve its expected value."
  }
}

run "invalid_engine" {
  command = plan

  variables {
    engine = "oracle"
  }

  expect_failures = [var.engine]
}
