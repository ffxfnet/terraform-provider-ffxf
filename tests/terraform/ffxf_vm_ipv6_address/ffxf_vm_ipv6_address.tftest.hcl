mock_provider "ffxf" {
  mock_resource "ffxf_vm_ipv6_address" {
    defaults = { id = "123/2001:db8::1" }
  }
}

run "ipv6" {
  command = apply
  assert {
    condition     = ffxf_vm_ipv6_address.test.id == "123/2001:db8::1"
    error_message = "expected imported-style ID"
  }
}
