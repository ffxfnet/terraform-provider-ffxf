mock_provider "ffxf" {
  mock_data "ffxf_oidc_providers" {
    defaults = { providers = [] }
  }
}

run "mock_oidc_providers" {
  command = apply
  assert {
    condition     = length(data.ffxf_oidc_providers.all.providers) == 0
    error_message = "The mock should return an empty list"
  }
}
