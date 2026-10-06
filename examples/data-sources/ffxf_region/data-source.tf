data "ffxf_region" "montreal" {
  slug = "montreal"
}

output "region_status" {
  value = data.ffxf_region.montreal.status
}

output "region_ipv6_available" {
  value = data.ffxf_region.montreal.ipv6
}