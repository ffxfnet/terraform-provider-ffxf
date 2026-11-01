terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

resource "ffxf_database_detach" "example" {
  database_id = 43
}
