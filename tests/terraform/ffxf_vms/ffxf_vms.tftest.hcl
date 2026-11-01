mock_provider "ffxf" {
  mock_data "ffxf_vms" {
    defaults = {
      vms = [{ id = 123, hostname = "vm-1", status = "running", plan = "nano", region = "montreal", image = "debian-13", ipv4 = "192.0.2.1", ipv6 = "2001:db8::1", billing_mode = "hourly" }]
    }
  }
}

run "vms" {
  command = plan
  assert {
    condition     = data.ffxf_vms.test.vms[0].id == 123
    error_message = "expected mocked VM"
  }
}
