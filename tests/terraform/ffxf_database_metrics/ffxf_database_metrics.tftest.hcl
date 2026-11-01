mock_provider "ffxf" {
  mock_data "ffxf_database_metrics" {
    defaults = { metrics_json = "{}" }
  }
}

run "ffxf_database_metrics_plan" {
  command = plan

  assert {
    condition     = data.ffxf_database_metrics.current.metrics_json == "{}"
    error_message = "The mocked ffxf_database_metrics did not return its expected value."
  }
}

run "invalid_metrics_range" {
  command = plan

  variables {
    metrics_range = "30d"
  }

  expect_failures = [var.metrics_range]
}
