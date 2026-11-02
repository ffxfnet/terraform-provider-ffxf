terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}
provider "ffxf" {}
variable "group_by" {
  type    = string
  default = "hour"
  validation {
    condition     = contains(["hour", "vm"], var.group_by)
    error_message = "invalid enum"
  }
}
data "ffxf_usage" "invalid" {
  group_by = var.group_by
}
