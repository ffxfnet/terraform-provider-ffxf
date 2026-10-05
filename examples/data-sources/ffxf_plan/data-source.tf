data "ffxf_plan" "nano" {
  slug = "nano"
}

output "plan_vcpu" {
  value = data.ffxf_plan.nano.vcpu
}

output "plan_prices" {
  value = data.ffxf_plan.nano.prices
}