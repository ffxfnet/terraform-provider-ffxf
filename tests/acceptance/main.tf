terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

variable "run_id" {
  type = string
}

variable "run_attempt" {
  type = string
}

data "ffxf_plan" "test" {
  slug = "nano"
}

data "ffxf_region" "test" {
  slug = "montreal"
}

data "ffxf_image" "test" {
  slug = "debian-13"
}

resource "ffxf_instance" "acceptance" {
  hostname = "gha-${var.run_id}-${var.run_attempt}"
  plan     = data.ffxf_plan.test.slug
  region   = data.ffxf_region.test.slug
  image    = data.ffxf_image.test.slug
  billing  = "hourly"
}
