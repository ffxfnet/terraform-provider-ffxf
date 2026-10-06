resource "ffxf_load_balancer" "web" {
  name = "website"
  vpc  = ffxf_vpc.production.id
}

resource "ffxf_load_balancer_pool" "frontend" {
  load_balancer_id = ffxf_load_balancer.web.id
  name             = "frontend"
  protocol         = "http"
  algorithm        = "roundrobin"
  health_type      = "http"
  health_path      = "/health"

  targets = [
    {
      vm   = 150
      port = 3000
    },
  ]
}

resource "ffxf_load_balancer_listener" "https" {
  load_balancer_id = ffxf_load_balancer.web.id
  port             = 443
  protocol         = "https"
  default_pool     = ffxf_load_balancer_pool.frontend.id
  redirect_https   = true

  rules = [
    {
      hostname = "app.example.com"
      pool     = ffxf_load_balancer_pool.frontend.id
    },
  ]
}