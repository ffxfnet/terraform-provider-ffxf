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

locals {
  name = "gha-${var.run_id}-${var.run_attempt}"
}

# Throwaway public key; its private half was never kept. Destroying it removes the key from the account.
resource "ffxf_ssh_key" "acceptance" {
  name       = local.name
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIK5qIyZWuJL9XZISbOW9e1D6+/dXw+U8ZLUW9asIhjT8 ffxf-acceptance"
}

resource "ffxf_vpc" "acceptance" {
  name   = local.name
  cidr   = "10.77.0.0/24"
  region = data.ffxf_region.test.slug
}

resource "ffxf_firewall" "acceptance" {
  name = local.name

  rules = [
    {
      protocol = "tcp"
      ports    = "22"
    },
    {
      protocol = "icmp"
    },
  ]
}

resource "ffxf_instance" "acceptance" {
  hostname = local.name
  plan     = data.ffxf_plan.test.slug
  region   = data.ffxf_region.test.slug
  image    = data.ffxf_image.test.slug
  billing  = "hourly"
  ssh_keys = [ffxf_ssh_key.acceptance.fingerprint]
}

resource "ffxf_firewall_member" "acceptance" {
  firewall_id = ffxf_firewall.acceptance.id
  vm_id       = ffxf_instance.acceptance.id
}

resource "ffxf_vpc_member" "acceptance" {
  vpc_id = ffxf_vpc.acceptance.id
  vm_id  = ffxf_instance.acceptance.id
}

resource "ffxf_vm_power_state" "acceptance" {
  vm_id  = ffxf_instance.acceptance.id
  state  = "running"
  method = "graceful"
}

data "ffxf_vm_network" "acceptance" {
  vm_id = ffxf_instance.acceptance.id

  depends_on = [ffxf_vpc_member.acceptance]
}

resource "ffxf_vm_ipv6_address" "acceptance" {
  vm_id   = ffxf_instance.acceptance.id
  address = cidrhost(strcontains(data.ffxf_vm_network.acceptance.ipv6_prefix, "/") ? data.ffxf_vm_network.acceptance.ipv6_prefix : "${data.ffxf_vm_network.acceptance.ipv6_prefix}/64", 66)
}

# ffxf_vm_rdns is not exercised: the API only accepts a hostname that already resolves to the address,
# which needs a DNS record the workflow does not own.

# Read-only lookups: these fail the run if the API shape drifts from the provider models.
data "ffxf_plans" "all" {
  region = data.ffxf_region.test.slug
}

data "ffxf_images" "linux" {
  region   = data.ffxf_region.test.slug
  plan     = data.ffxf_plan.test.slug
  category = "linux"
}

data "ffxf_vpcs" "all" {
  depends_on = [ffxf_vpc.acceptance]
}

data "ffxf_ssh_keys" "all" {
  depends_on = [ffxf_ssh_key.acceptance]
}

data "ffxf_vms" "all" {
  depends_on = [ffxf_instance.acceptance]
}

data "ffxf_vm_metrics" "acceptance" {
  vm_id     = ffxf_instance.acceptance.id
  timeframe = "hour"
}

data "ffxf_vm_backups" "acceptance" {
  vm_id = ffxf_instance.acceptance.id
}

data "ffxf_vm_actions" "acceptance" {
  vm_id = ffxf_instance.acceptance.id

  depends_on = [ffxf_vm_power_state.acceptance]
}

check "acceptance_outputs" {
  assert {
    condition     = ffxf_instance.acceptance.ipv4 != null && length(data.ffxf_vms.all.vms) > 0
    error_message = "The acceptance instance was not returned by the VM inventory."
  }

  assert {
    condition     = length(data.ffxf_ssh_keys.all.ssh_keys) > 0 && length(data.ffxf_vpcs.all.vpcs) > 0
    error_message = "The acceptance SSH key or VPC is missing from the inventories."
  }
}
