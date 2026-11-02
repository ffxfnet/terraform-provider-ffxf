terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

variable "run_id" {
  type = string
}

variable "run_attempt" {
  type = string
}

locals {
  name = "gha-${var.run_id}-${var.run_attempt}"
}

# Everything here is new: a credential cannot modify the role or policy it holds, so the
# workflow's own access is never touched.
resource "ffxf_policy" "acceptance" {
  name        = local.name
  description = "Acceptance test policy"

  document = jsonencode({
    version = "2026-10-01"
    statements = [{
      effect    = "allow"
      actions   = ["vm:read"]
      resources = ["*"]
    }]
  })
}

resource "ffxf_role" "acceptance" {
  name        = local.name
  description = "Acceptance test role"
  policy_ids  = [ffxf_policy.acceptance.id]
}

resource "ffxf_api_key" "acceptance" {
  name       = local.name
  role       = ffxf_role.acceptance.name
  expires_at = "2099-12-31"
}

data "ffxf_oidc_providers" "all" {}

# The subject names a branch that never exists, so the binding can never authenticate anyone.
resource "ffxf_oidc_binding" "acceptance" {
  provider_id = data.ffxf_oidc_providers.all.providers[0].id
  name        = local.name
  role        = ffxf_role.acceptance.name

  claim_conditions = [
    {
      claim = "sub"
      value = "repo:ffxfnet/terraform-provider-ffxf:ref:refs/heads/acceptance-never-exists"
    },
  ]
}

data "ffxf_policies" "all" {
  depends_on = [ffxf_policy.acceptance]
}

data "ffxf_roles" "all" {
  depends_on = [ffxf_role.acceptance]
}

data "ffxf_api_keys" "all" {
  depends_on = [ffxf_api_key.acceptance]
}

check "access_objects_listed" {
  assert {
    condition = (
      contains([for p in data.ffxf_policies.all.policies : p.name], local.name) &&
      contains([for r in data.ffxf_roles.all.roles : r.name], local.name) &&
      contains([for k in data.ffxf_api_keys.all.api_keys : k.name], local.name)
    )
    error_message = "The created policy, role or API key is missing from the listings."
  }
}

check "api_key_secret_returned" {
  assert {
    condition     = startswith(ffxf_api_key.acceptance.token, "ffxf_")
    error_message = "The API key secret was not returned at creation."
  }
}
