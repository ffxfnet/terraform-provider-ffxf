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

data "ffxf_region" "test" {
  slug = "montreal"
}

data "ffxf_plan" "test" {
  slug = "nano"
}

data "ffxf_image" "test" {
  slug = "debian-13"
}

locals {
  name = "gha-${var.run_id}-${var.run_attempt}"
}

resource "ffxf_vpc" "acceptance" {
  name   = local.name
  cidr   = "10.79.0.0/24"
  region = data.ffxf_region.test.slug
}

resource "ffxf_load_balancer" "acceptance" {
  name = local.name
  vpc  = ffxf_vpc.acceptance.id
}

# A backend VM must belong to the load balancer's VPC before it can be a pool target.
resource "ffxf_instance" "acceptance" {
  hostname = local.name
  plan     = data.ffxf_plan.test.slug
  region   = data.ffxf_region.test.slug
  image    = data.ffxf_image.test.slug
  billing  = "hourly"
}

resource "ffxf_vpc_member" "acceptance" {
  vpc_id = ffxf_vpc.acceptance.id
  vm_id  = ffxf_instance.acceptance.id
}

resource "ffxf_load_balancer_pool" "acceptance" {
  load_balancer_id = ffxf_load_balancer.acceptance.id
  name             = "acceptance"
  protocol         = "http"
  algorithm        = "roundrobin"
  health_type      = "tcp"

  targets = [
    {
      vm   = ffxf_vpc_member.acceptance.vm_id
      port = 8080
    },
  ]
}

resource "ffxf_load_balancer_listener" "acceptance" {
  load_balancer_id = ffxf_load_balancer.acceptance.id
  port             = 80
  protocol         = "http"
  default_pool     = ffxf_load_balancer_pool.acceptance.id
}

check "pool_has_target" {
  assert {
    condition     = length(ffxf_load_balancer_pool.acceptance.targets) == 1 && ffxf_load_balancer_pool.acceptance.targets[0].address != ""
    error_message = "The VM was not attached to the pool as a target."
  }
}
