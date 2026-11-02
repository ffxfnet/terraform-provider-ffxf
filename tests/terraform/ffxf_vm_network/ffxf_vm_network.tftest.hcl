mock_provider "ffxf" {
  mock_data "ffxf_vm_network" {
    defaults = { port_mbps = 100, ipv4 = [], ipv6 = [] }
  }
}

run "network" {
  command = plan
  assert {
    condition     = data.ffxf_vm_network.test.port_mbps == 100
    error_message = "expected network output"
  }
}
