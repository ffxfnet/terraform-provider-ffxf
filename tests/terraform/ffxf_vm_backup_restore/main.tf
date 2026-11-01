terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

resource "ffxf_vm_backup_restore" "test" {
  vm_id  = 123
  backup = "backup-1"
}
