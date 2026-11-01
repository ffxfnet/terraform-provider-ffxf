terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}
provider "ffxf" {}
variable "category" {
  type    = string
  default = "linux"
  validation {
    condition     = contains(["linux", "windows", "bsd", "app", "other"], var.category)
    error_message = "invalid enum"
  }
}
data "ffxf_images" "invalid" {
  category = var.category
}
