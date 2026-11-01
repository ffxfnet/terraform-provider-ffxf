resource "ffxf_vm_power_state" "example" {
  vm_id  = 123
  state  = "running"
  method = "graceful"
}
