terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

data "ffxf_runner_jobs" "repo" {
  repository = "owner/repo"
  status     = "queued"
  limit      = 25
}
