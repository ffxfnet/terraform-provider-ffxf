data "ffxf_load_balancer_metrics" "web" {
  load_balancer_id = ffxf_load_balancer.web.id
  range            = "24h"
}

output "requests_last_day" {
  value = data.ffxf_load_balancer_metrics.web.requests
}