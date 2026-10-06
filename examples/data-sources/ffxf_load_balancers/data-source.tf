data "ffxf_load_balancers" "all" {}

output "load_balancer_addresses" {
  value = [for lb in data.ffxf_load_balancers.all.load_balancers : lb.ipv4]
}

output "load_balancer_quota" {
  value = data.ffxf_load_balancers.all.quota
}