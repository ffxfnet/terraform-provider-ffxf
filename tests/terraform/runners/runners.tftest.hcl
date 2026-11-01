mock_provider "ffxf" {
  mock_data "ffxf_runners" {
    defaults = { runners = { sizes = [], concurrency = { max_vcpu = 0, running_vcpu = 0 }, installations = [], month = { period_start = "2026-01-01T00:00:00Z", minutes = 0, free_minutes_used = 0, free_minutes_allowance = 0, amount = "0", cache_amount = "0", currency = "CAD", repositories = [] }, cache = { price_per_gb_month = "0", currency = "CAD", bytes = 0, repository_limit_gb_max = 0, retention_days = 0, repositories = [] }, docker_cache = { repository_size_gb = 0, retention_days = 0, repositories = [] } } }
  }
}

run "mock_runners" {
  command = apply
  assert {
    condition = length(data.ffxf_runners.current.runners.sizes) == 0
    error_message = "runner sizes should be empty in the mock"
  }
}
