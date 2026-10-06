resource "ffxf_firewall" "web" {
  name = "web"

  rules = [
    {
      protocol = "tcp"
      ports    = "22"
      sources  = ["203.0.113.0/24"]
    },
    {
      protocol = "tcp"
      ports    = "80,443"
    },
    {
      protocol = "icmp"
    },
  ]
}

resource "ffxf_firewall_member" "web" {
  firewall_id = ffxf_firewall.web.id
  vm_id       = ffxf_instance.web.id
}