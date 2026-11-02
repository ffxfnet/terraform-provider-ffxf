terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

resource "ffxf_policy" "readers" {
  name        = "vm-readers"
  description = "Read-only access to virtual machines"

  document = jsonencode({
    version = "2026-10-01"
    statements = [{
      effect    = "allow"
      actions   = ["vm:read"]
      resources = ["*"]
    }]
  })
}

resource "ffxf_role" "readers" {
  name       = "readers"
  policy_ids = [ffxf_policy.readers.id]
}
