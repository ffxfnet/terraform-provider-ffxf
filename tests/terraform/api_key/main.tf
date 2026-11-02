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

resource "ffxf_api_key" "ci" {
  name       = "ci"
  role       = ffxf_role.readers.name
  expires_at = "2030-12-31"
}

output "ci_api_key" {
  value     = ffxf_api_key.ci.token
  sensitive = true
}
