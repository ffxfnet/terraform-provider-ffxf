# terraform-provider-ffxf

Manage FFXF Cloud virtual machines with OpenTofu. Define instance configuration in HCL and let OpenTofu create, update, refresh, and destroy the resources to match your configuration.

The provider currently supports the `ffxf_instance` resource. It creates a VM from a plan, region, image, and hostname, tracks its assigned ID and IPv4 address, and updates hostname changes in place. Changes to plan, region, image, or billing require the instance to be replaced.

## Configuration

Provide an FFXF Cloud API token in the provider configuration or set the `FFXF_TOKEN` environment variable. The API endpoint is optional; it defaults to `https://api.ffxf.net/v1`. Set `FFXF_ENDPOINT` to override it through the environment.

```hcl
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
	# endpoint = "https://api.ffxf.net/v1"
}

resource "ffxf_instance" "web" {
	hostname = "web-server-01"
	plan     = "nano"
	region   = "montreal"
	image    = "debian-13"
	billing  = "hourly"
}
```

The `billing` argument is optional and defaults to `hourly`. The provider also accepts `FFXF_TOKEN` instead of setting `token` in the provider block. Mark the variable holding your token as sensitive and avoid committing credentials to version control.
