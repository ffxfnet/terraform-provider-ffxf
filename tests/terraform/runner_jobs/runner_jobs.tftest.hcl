mock_provider "ffxf" {
  mock_data "ffxf_runner_jobs" {
    defaults = { jobs = [] }
  }
}

run "mock_runner_jobs" {
  command = apply
  assert {
    condition = length(data.ffxf_runner_jobs.repo.jobs) == 0
    error_message = "runner jobs should be empty in the mock"
  }
}
