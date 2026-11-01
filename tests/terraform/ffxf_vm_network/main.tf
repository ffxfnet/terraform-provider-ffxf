terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

data "ffxf_vm_network" "test" {
  vm_id = 123
}
