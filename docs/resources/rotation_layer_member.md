# incidentrelay_rotation_layer_member

Manages a member period in a rotation layer.

```hcl
resource "incidentrelay_rotation_layer_member" "alice" {
  layer_id = incidentrelay_rotation_layer.business_hours.id
  user_id  = incidentrelay_admin_user.alice.id
  position = 0
}
```

