terraform {
  required_providers {
    ffxf = { source = "ffxfnet/ffxf" }
  }
}

provider "ffxf" {}

variable "engine" {
  type    = string
  default = "postgresql"

  validation {
    condition     = contains(["postgresql", "mysql", "mariadb", "valkey"], var.engine)
    error_message = "Choose a supported database engine."
  }
}

resource "ffxf_database" "example" {
  name   = "orders"
  vpc    = 123
  engine = var.engine
}
