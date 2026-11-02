resource "ffxf_api_key" "ci" {
  name       = "ci"
  role       = "readers"
  expires_at = "2030-12-31"
}

output "ci_api_key" {
  value     = ffxf_api_key.ci.token
  sensitive = true
}
