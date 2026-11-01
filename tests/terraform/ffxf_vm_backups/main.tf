terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

data "ffxf_vm_backups" "test" {
  vm_id = 123
}
