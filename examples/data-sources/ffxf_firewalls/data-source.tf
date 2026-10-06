data "ffxf_firewalls" "all" {}

output "firewall_ids" {
  value = [for firewall in data.ffxf_firewalls.all.firewalls : firewall.id]
}

output "firewall_quota" {
  value = data.ffxf_firewalls.all.quota
}