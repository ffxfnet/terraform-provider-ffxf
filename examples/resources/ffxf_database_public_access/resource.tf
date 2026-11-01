resource "ffxf_database_public_access" "example" {
  database_id = 42
  enabled     = true
  sources     = ["203.0.113.10/32"]
}
