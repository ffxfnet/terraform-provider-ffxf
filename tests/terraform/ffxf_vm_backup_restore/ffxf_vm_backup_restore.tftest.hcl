mock_provider "ffxf" {}

run "restore" {
  command = apply
  assert {
    condition     = ffxf_vm_backup_restore.test.backup == "backup-1"
    error_message = "expected selected backup"
  }
}
