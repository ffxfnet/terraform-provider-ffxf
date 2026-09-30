terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {
  # The token can also be set by env variable using FFXF_TOKEN
}