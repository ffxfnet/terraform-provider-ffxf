terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

data "ffxf_account" "current" {}

data "ffxf_usage" "current" {
  group_by = "vm"
}

data "ffxf_plans" "all" {}

data "ffxf_images" "linux" {
  category = "linux"
}

check "catalog_not_empty" {
  assert {
    condition     = length(data.ffxf_plans.all.plans) > 0 && length(data.ffxf_images.linux.images) > 0
    error_message = "The plan or image catalog is empty."
  }
}
