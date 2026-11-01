mock_provider "ffxf" {
  mock_data "ffxf_images" {
    defaults = {}
  }
}
run "expect_invalid_category" {
  command = plan
  variables {
    category = "unknown"
  }
  expect_failures = [var.category]
}
