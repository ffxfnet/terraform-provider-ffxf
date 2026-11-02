mock_provider "ffxf" {
  mock_data "ffxf_api_keys" {
    defaults = { api_keys = [] }
  }
}

run "mock_api_keys" {
  command = apply
  assert {
    condition     = length(data.ffxf_api_keys.all.api_keys) == 0
    error_message = "The mock should return an empty list"
  }
}
