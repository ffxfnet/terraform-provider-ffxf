terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

resource "ffxf_vm_rdns" "test" {
  vm_id    = 123
  address  = "2001:db8::1"
  hostname = "vm.example.net"
}
