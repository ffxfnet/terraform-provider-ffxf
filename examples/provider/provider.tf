terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

variable "ffxf_token" {
  type      = string
  sensitive = true
}

provider "ffxf" {
  token = var.ffxf_token
  # Alternatively, set FFXF_TOKEN in the environment and omit token here.
  # endpoint = "https://api.ffxf.net/v1"
}