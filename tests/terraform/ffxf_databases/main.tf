terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

data "ffxf_databases" "all" {}
