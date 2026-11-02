terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

resource "ffxf_vm_ipv6_address" "test" {
  vm_id   = 123
  address = "2001:db8::1"
}
