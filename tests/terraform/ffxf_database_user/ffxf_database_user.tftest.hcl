mock_provider "ffxf" {
  mock_resource "ffxf_database_user" {
    defaults = { id = "42/app", password = "mock-password" }
  }
}

run "ffxf_database_user_apply" {
  command = apply

  assert {
    condition     = ffxf_database_user.example.id == "42/app"
    error_message = "The mocked ffxf_database_user did not preserve its expected value."
  }
}
