resource "ffxf_database_fork" "example" {
  source_database_id = 42
  name               = "orders-staging"
}
