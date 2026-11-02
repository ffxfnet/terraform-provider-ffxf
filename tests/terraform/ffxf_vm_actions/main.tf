terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

data "ffxf_vm_actions" "test" {
  status = "error"
}
