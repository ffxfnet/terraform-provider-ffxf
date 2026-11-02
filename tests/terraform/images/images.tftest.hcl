mock_provider "ffxf" {
  mock_data "ffxf_images" {
    defaults = { images = [] }
  }
}

run "mock_images" {
  command = apply
  assert {
    condition     = length(data.ffxf_images.catalog.images) == 0
    error_message = "images should be empty in the mock"
  }
}
