mock_provider "ffxf" {}

run "power" {
  command = apply
  assert {
    condition     = var.state == "running"
    error_message = "expected running state"
  }
}

run "invalid-state" {
  command = plan
  variables { state = "invalid" }
  expect_failures = [var.state]
}
