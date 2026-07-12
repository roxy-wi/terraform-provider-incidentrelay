# incidentrelay_business_service_component

Manages a technical service component inside a business service.

```hcl
resource "incidentrelay_business_service_component" "api" {
  business_service_id = incidentrelay_business_service.checkout.id
  service_id          = incidentrelay_service.api.id
  criticality         = "required"
  impact_weight       = 100
}
```

