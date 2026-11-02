terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

resource "ffxf_vm_reinstall" "test" {
  vm_id    = 123
  image    = "debian-13"
  ssh_keys = ["key-id"]
}
