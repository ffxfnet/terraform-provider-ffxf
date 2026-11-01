resource "ffxf_database_restore" "example" {
  source_database_id = 42
  name               = "orders-before-migration"
  at                 = "2026-10-14T09:30:00Z"
}
