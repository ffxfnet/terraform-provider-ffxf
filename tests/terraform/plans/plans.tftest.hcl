mock_provider "ffxf" {
  mock_data "ffxf_plans" {
    defaults = { plans = [] }
  }
}

run "mock_plans" {
  command = apply
  assert {
    condition = length(data.ffxf_plans.catalog.plans) == 0
    error_message = "plans should be empty in the mock"
  }
}
