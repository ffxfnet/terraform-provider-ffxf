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
  cidr   = "10.78.0.0/24"
  region = data.ffxf_region.test.slug
}

data "ffxf_database_options" "catalog" {}

resource "ffxf_database" "acceptance" {
  name   = local.name
  vpc    = ffxf_vpc.acceptance.id
  engine = "postgresql"
}

resource "ffxf_database_user" "acceptance" {
  database_id = ffxf_database.acceptance.id
  username    = "acceptance"
}

resource "ffxf_database_logical_database" "acceptance" {
  database_id = ffxf_database.acceptance.id
  name        = "acceptance"
}

data "ffxf_database_credentials" "acceptance" {
  database_id = ffxf_database.acceptance.id

  depends_on = [ffxf_database_user.acceptance]
}

data "ffxf_databases" "all" {
  depends_on = [ffxf_database.acceptance]
}
