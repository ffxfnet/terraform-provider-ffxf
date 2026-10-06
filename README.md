# terraform-provider-ffxf

Manage FFXF Cloud virtual machines with OpenTofu. Define instance configuration in HCL and let OpenTofu create, update, refresh, and destroy the resources to match your configuration.

The provider supports `ffxf_instance`, `ffxf_vpc`, `ffxf_firewall`, `ffxf_firewall_member`, and the `ffxf_load_balancer`, `ffxf_load_balancer_pool`, and `ffxf_load_balancer_listener` resources. Inventory data sources list firewalls and load balancers with account quotas; `ffxf_load_balancer_metrics` reads request, traffic, connection, and session statistics. Read-only catalog data sources are available for regions, plans, and images.

Use firewalls to manage inbound and outbound rules, then attach them to machines with `ffxf_firewall_member`. Load balancers are attached to a private network and configured with pools (including their complete target sets) and listeners (including their complete routing rules). HTTPS certificates are managed by FFXF Cloud. The instance resource tracks its assigned ID and IPv4 address, and applies hostname changes in place. Changes to plan, region, image, or billing replace the instance; changing a load balancer's VPC also requires replacement.

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
