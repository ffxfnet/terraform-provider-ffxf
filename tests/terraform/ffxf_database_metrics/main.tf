terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

variable "metrics_range" {
  type    = string
  default = "1h"

  validation {
    condition     = contains(["1h", "24h", "7d"], var.metrics_range)
    error_message = "Choose a supported metrics range."
  }
}

data "ffxf_database_metrics" "current" {
  database_id = 42
  range       = var.metrics_range
}
