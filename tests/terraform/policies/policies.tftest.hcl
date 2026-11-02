mock_provider "ffxf" {
  mock_data "ffxf_policies" {
    defaults = { policies = [] }
  }
}

run "mock_policies" {
  command = apply
  assert {
    condition     = length(data.ffxf_policies.all.policies) == 0
    error_message = "The mock should return an empty list"
  }
}
