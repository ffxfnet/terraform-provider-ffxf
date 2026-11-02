mock_provider "ffxf" {
  mock_data "ffxf_vpcs" {
    defaults = { quota = 3, vpcs = [] }
  }
}

run "mock_vpcs" {
  command = apply
  assert {
    condition     = data.ffxf_vpcs.all.quota == 3
    error_message = "quota should come from the mock"
  }
}
