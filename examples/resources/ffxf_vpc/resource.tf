resource "ffxf_vpc" "production" {
  name   = "production"
  cidr   = "10.0.0.0/24"
  region = "montreal"
}