mock_provider "ffxf" {
  mock_data "ffxf_vm_actions" {
    defaults = { actions = [] }
  }
}

run "actions" {
  command = plan
  assert {
    condition     = length(data.ffxf_vm_actions.test.actions) == 0
    error_message = "expected empty action list"
  }
}
