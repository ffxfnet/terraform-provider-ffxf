mock_provider "ffxf" {
  mock_data "ffxf_roles" {
    defaults = { roles = [] }
  }
}

run "mock_roles" {
  command = apply
  assert {
    condition     = length(data.ffxf_roles.all.roles) == 0
    error_message = "The mock should return an empty list"
  }
}
