terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

data "ffxf_runners" "current" {}

data "ffxf_runner_jobs" "current" {}
