resource "ffxf_instance" "example" {
  hostname = "web-server-01"
  plan     = "nano"
  region   = "montreal"
  image    = "debian-13"
  billing  = "hourly"
}