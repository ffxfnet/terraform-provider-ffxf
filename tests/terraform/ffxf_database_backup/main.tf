terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

resource "ffxf_database_backup" "example" {
  database_id = 42
}
