resource "ffxf_vm_rdns" "example" {
  vm_id    = 123
  address  = "2001:db8::42"
  hostname = "vm.example.net"
}
