data "ffxf_oidc_providers" "all" {}

resource "ffxf_oidc_binding" "github" {
  provider_id = data.ffxf_oidc_providers.all.providers[0].id
  name        = "github-actions"
  role        = "readers"

  claim_conditions = [
    {
      claim = "sub"
      value = "repo:acme/infrastructure:ref:refs/heads/main"
    },
  ]
}
