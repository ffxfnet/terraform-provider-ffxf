terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

resource "ffxf_database_logical_database" "example" {
  database_id = 42
  name        = "analytics"
}
