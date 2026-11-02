mock_provider "ffxf" {
  mock_resource "ffxf_instance" {
    defaults = { id = 123, ipv4 = "192.0.2.1" }
  }
}

run "instance-ssh-keys" {
  command = apply
  assert {
    condition     = toset(ffxf_instance.test.ssh_keys) == toset(["SHA256:key"])
    error_message = "expected SSH keys in config"
  }
}

run "invalid-destroy-when" {
  command = plan
  variables { destroy_when = "later" }
  expect_failures = [var.destroy_when]
}
