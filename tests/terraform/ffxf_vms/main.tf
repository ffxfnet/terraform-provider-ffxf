terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

data "ffxf_vms" "test" {
  status = "running"
}
