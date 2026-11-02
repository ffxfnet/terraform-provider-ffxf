terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

variable "destroy_when" {
  type    = string
  default = "now"
  validation {
    condition     = contains(["now", "period_end"], var.destroy_when)
    error_message = "destroy_when must be now or period_end"
  }
}

resource "ffxf_instance" "test" {
  hostname     = "vm.example.net"
  plan         = "nano"
  region       = "montreal"
  image        = "debian-13"
  ssh_keys     = ["SHA256:key"]
  destroy_when = var.destroy_when
}
