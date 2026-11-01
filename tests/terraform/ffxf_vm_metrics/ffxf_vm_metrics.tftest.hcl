mock_provider "ffxf" {
  mock_data "ffxf_vm_metrics" {
    defaults = { step_seconds = 60, series = [] }
  }
}

run "metrics" {
  command = plan
  assert {
    condition     = data.ffxf_vm_metrics.test.step_seconds == 60
    error_message = "expected metric step"
  }
}
