terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

data "ffxf_usage" "month" {
  group_by = "hour"
}

