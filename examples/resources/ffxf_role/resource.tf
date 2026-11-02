resource "ffxf_role" "readers" {
  name        = "readers"
  description = "Can read virtual machines"
  policy_ids  = [ffxf_policy.readers.id]
}
