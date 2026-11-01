terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

data "ffxf_vm_metrics" "test" {
  vm_id     = 123
  timeframe = "hour"
}
