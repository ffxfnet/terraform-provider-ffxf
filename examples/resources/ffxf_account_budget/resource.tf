terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

resource "ffxf_account_budget" "monthly" {
  monthly = "150.00"
}
