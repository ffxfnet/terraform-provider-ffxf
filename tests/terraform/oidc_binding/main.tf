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

data "ffxf_oidc_providers" "all" {}

resource "ffxf_oidc_binding" "github" {
  provider_id = data.ffxf_oidc_providers.all.providers[0].id
  name        = "github-actions"
  role        = ffxf_role.readers.name

  claim_conditions = [
    {
      claim = "sub"
      value = "repo:acme/infrastructure:ref:refs/heads/main"
    },
  ]
}
