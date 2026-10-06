# terraform-provider-ffxf

Manage FFXF Cloud virtual machines with OpenTofu. Define instance configuration in HCL and let OpenTofu create, update, refresh, and destroy the resources to match your configuration.

The provider supports the `ffxf_instance` and `ffxf_vpc` resources, plus read-only catalog data sources for regions, plans, and images. Use the data sources to inspect the offerings available to your account before creating an instance. The instance resource tracks its assigned ID and IPv4 address, and updates hostname changes in place. Changes to plan, region, image, or billing require the instance to be replaced. VPCs are created with a name, private IPv4 range, and region; changes to these settings replace the network.

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

data "ffxf_plan" "nano" {
	slug = "nano"
}

data "ffxf_image" "debian" {
	slug = "debian-13"
}

data "ffxf_region" "montreal" {
	slug = "montreal"
}
```

The `billing` argument is optional and defaults to `hourly`. The provider also accepts `FFXF_TOKEN` instead of setting `token` in the provider block. Mark the variable holding your token as sensitive and avoid committing credentials to version control.
