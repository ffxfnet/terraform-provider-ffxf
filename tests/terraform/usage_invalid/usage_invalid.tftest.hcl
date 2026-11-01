mock_provider "ffxf" {
  mock_data "ffxf_usage" {
    defaults = {}
  }
}
run "expect_invalid_group_by" {
  command = plan
  variables {
    group_by = "day"
  }
  expect_failures = [var.group_by]
}
