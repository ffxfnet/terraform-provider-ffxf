mock_provider "ffxf" {}

run "reinstall" {
  command = apply
  assert {
    condition     = ffxf_vm_reinstall.test.image == "debian-13"
    error_message = "expected selected image"
  }
}
