# incidentrelay_service_dependency

Manages an upstream service dependency.

```hcl
resource "incidentrelay_service_dependency" "database" {
  service_id             = incidentrelay_service.api.id
  depends_on_service_id  = incidentrelay_service.database.id
  dependency_type        = "hard"
  criticality            = "required"
}
```

