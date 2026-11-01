mock_provider "ffxf" {
  mock_data "ffxf_account" {
    defaults = { account = { currency = "CAD", billing_mode = "prepaid", credit = { balance = "0", paid = "0", gifted = "0", gifted_expires_at = null }, hourly = { burn_rate = "0", runway_hours = null, active_machines = 0 }, budget = { monthly = null, month_to_date = "0", percent = null, cap_percent = 100 }, quota = { limit = 0, used = 0, remaining = 0 }, auto_topup = { enabled = false, threshold = null, amount = null } } }
  }
}

run "mock_account" {
  command = apply
  assert {
    condition = data.ffxf_account.current.account.currency == "CAD"
    error_message = "account currency should come from the mock"
  }
}
