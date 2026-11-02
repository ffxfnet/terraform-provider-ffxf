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
