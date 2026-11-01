mock_provider "ffxf" {}

run "rdns" {
  command = apply
  assert {
    condition     = ffxf_vm_rdns.test.hostname == "vm.example.net"
    error_message = "expected rDNS hostname"
  }
}
