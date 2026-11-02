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

data "ffxf_plan" "first" {
  slug = data.ffxf_plans.all.plans[0].slug
}

data "ffxf_vms" "all" {}

data "ffxf_ssh_keys" "all" {}

data "ffxf_vpcs" "all" {}

data "ffxf_firewalls" "all" {}

data "ffxf_load_balancers" "all" {}

data "ffxf_policies" "all" {}

data "ffxf_roles" "all" {}

data "ffxf_api_keys" "all" {}

data "ffxf_oidc_providers" "all" {}

check "lookups_consistent" {
  assert {
    condition     = data.ffxf_plan.first.slug == data.ffxf_plans.all.plans[0].slug
    error_message = "Looking a plan up by slug returned a different plan."
  }
}

check "access_lists_readable" {
  assert {
    condition = (
      length(data.ffxf_oidc_providers.all.providers) > 0 &&
      length(data.ffxf_roles.all.roles) > 0 &&
      length(data.ffxf_policies.all.policies) > 0
    )
    error_message = "The OIDC provider, role or policy listing is empty although this job is authenticated through them."
  }
}
