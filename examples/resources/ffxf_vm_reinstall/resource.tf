resource "ffxf_vm_reinstall" "example" {
  vm_id    = 123
  image    = "debian-13"
  ssh_keys = ["SHA256:example"]
}
