terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

variable "state" {
  type    = string
  default = "running"
  validation {
    condition     = contains(["running", "stopped"], var.state)
    error_message = "state must be running or stopped"
  }
}

resource "ffxf_vm_power_state" "test" {
  vm_id = 123
  state = var.state
}
