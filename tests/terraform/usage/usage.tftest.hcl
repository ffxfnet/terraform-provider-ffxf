mock_provider "ffxf" {
  mock_data "ffxf_usage" {
    defaults = { records = [], total = "0.00", currency = "CAD" }
  }
}

run "mock_usage" {
  command = apply
  assert {
    condition = data.ffxf_usage.month.currency == "CAD"
    error_message = "usage currency should come from the mock"
  }
}
