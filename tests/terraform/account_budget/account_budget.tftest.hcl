mock_provider "ffxf" {
  mock_resource "ffxf_account_budget" {
    defaults = { id = "account-budget", month_to_date = "0.00", percent = 0, cap_percent = 100 }
  }
}

run "mock_account_budget" {
  command = apply
  assert {
    condition     = ffxf_account_budget.monthly.monthly == "150.00"
    error_message = "monthly budget should match configuration"
  }
}
