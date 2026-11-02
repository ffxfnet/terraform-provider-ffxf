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

# The pool has no targets: attaching one would require a billed VM in the VPC.
resource "ffxf_load_balancer_pool" "acceptance" {
  load_balancer_id = ffxf_load_balancer.acceptance.id
  name             = "acceptance"
  protocol         = "http"
  algorithm        = "roundrobin"
  health_type      = "tcp"
}

resource "ffxf_load_balancer_listener" "acceptance" {
  load_balancer_id = ffxf_load_balancer.acceptance.id
  port             = 80
  protocol         = "http"
  default_pool     = ffxf_load_balancer_pool.acceptance.id
}
