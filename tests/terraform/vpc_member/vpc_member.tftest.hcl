mock_provider "ffxf" {
  mock_resource "ffxf_vpc_member" {
    defaults = { id = "12-34", hostname = "web", ip = "10.0.0.2", configuration = "automatic" }
  }
}

run "mock_vpc_member" {
  command = apply
  assert {
    condition     = ffxf_vpc_member.web.vpc_id == 12
    error_message = "VPC ID should match configuration"
  }
}
