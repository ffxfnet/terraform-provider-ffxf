mock_provider "ffxf" {
  mock_data "ffxf_vm_backups" {
    defaults = { automatic_enabled = true, retention = 7, backups = [] }
  }
}

run "backups" {
  command = plan
  assert {
    condition     = data.ffxf_vm_backups.test.automatic_enabled
    error_message = "expected automatic backups"
  }
}
