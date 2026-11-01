terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

data "ffxf_database_credentials" "current" {
  database_id = 42
}
