terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

data "ffxf_images" "catalog" {
  region = "montreal"
  plan = "nano"
  category = "linux"
}

