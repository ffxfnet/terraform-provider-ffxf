data "ffxf_plan" "selected" {
  slug = "nano"
}

data "ffxf_region" "selected" {
  slug = "montreal"
}

data "ffxf_image" "selected" {
  slug = "debian-13"
}

resource "ffxf_instance" "example" {
  hostname = "web-server-01"
  plan     = data.ffxf_plan.selected.slug
  region   = data.ffxf_region.selected.slug
  image    = data.ffxf_image.selected.slug
  billing  = "hourly"
}