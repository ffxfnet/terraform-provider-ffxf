terraform {
  required_providers {
    ffxf = {
      source = "ffxfnet/ffxf"
    }
  }
}

provider "ffxf" {}

resource "ffxf_ssh_key" "work" {
  name       = "work"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePublicKey user@example"
}
