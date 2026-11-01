mock_provider "ffxf" {
  mock_data "ffxf_ssh_keys" {
    defaults = { ssh_keys = [] }
  }
}

run "mock_ssh_keys" {
  command = apply
  assert {
    condition = length(data.ffxf_ssh_keys.all.ssh_keys) == 0
    error_message = "SSH keys should be empty in the mock"
  }
}
