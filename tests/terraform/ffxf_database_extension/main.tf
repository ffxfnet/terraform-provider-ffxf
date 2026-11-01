terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

resource "ffxf_database_extension" "example" {
  database_id = 42
  name        = "defaultdb"
  extension   = "vector"
}
