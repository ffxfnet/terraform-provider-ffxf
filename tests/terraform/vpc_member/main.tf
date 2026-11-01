terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

resource "ffxf_vpc_member" "web" {
  vpc_id = 12
  vm_id = 34
}
