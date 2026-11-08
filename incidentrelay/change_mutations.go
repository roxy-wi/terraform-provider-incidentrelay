package incidentrelay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func (k changeKind) create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Client
	if !d.Get("active").(bool) {
		return diag.Errorf("IR only creates active Change objects; refusing to temporarily enable an inactive %s", k)
	}
	if k == changeIntegration {
		body := map[string]interface{}{"name": d.Get("name"), "owner_team_id": d.Get("owner_team_id")}
		if group, ok := d.GetOk("owner_group_id"); ok {
			body["owner_group_id"] = group
		}
		r, err := changeRequest(ctx, client, http.MethodPost, changeAdminPath, body, 0)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(strconv.Itoa(r.ID))
		return diag.FromErr(k.save(d, r))
	}

	client.changeWrite.Lock()
	defer client.changeWrite.Unlock()
	parent := d.Get("integration_id").(int)
	base := integrationPath(parent)
	before, err := changeRequest(ctx, client, http.MethodGet, base, nil, parent)
	if err != nil {
		return diag.FromErr(err)
	}
	body := map[string]interface{}{"expected_revision": before.Revision}
	path := base + "/sources"
	if k == changeSource {
		for _, field := range []string{"owner_team_id", "source", "ordering_mode"} {
			body[field] = d.Get(field)
		}
	} else {
		path += fmt.Sprintf("/%d/grants", d.Get("source_id"))
		for _, field := range []string{"target_team_id", "capability"} {
			body[field] = d.Get(field)
		}
		if v, ok := d.GetOk("target_service_id"); ok {
			body["target_service_id"] = v
		}
	}
	after, err := changeRequest(ctx, client, http.MethodPost, path, body, parent)
	if err != nil {
		return diag.FromErr(err)
	}
	// The response ID is the integration's ID. Identify the created child by
	// its server-enforced unique key within this parent, not by array position.
	for _, s := range after.Sources {
		if k == changeSource && s.Team == d.Get("owner_team_id").(int) && s.Name == d.Get("source").(string) {
			d.SetId(strconv.Itoa(s.ID))
			return diag.FromErr(k.save(d, after))
		}
		if k == changeGrant && s.ID == d.Get("source_id").(int) {
			for _, g := range s.Grants {
				if g.Team == d.Get("target_team_id").(int) && g.Service == d.Get("target_service_id").(int) && g.Capability == d.Get("capability").(string) {
					d.SetId(strconv.Itoa(g.ID))
					return diag.FromErr(k.save(d, after))
				}
			}
		}
	}
	return diag.Errorf("IR accepted creation but the returned detail does not contain the new %s; inspect the integration and import the object before retrying", k)
}

// Locate the current parent source by grant ID: moving a grant uses the old
// URL and sends the new source_id in PATCH. Reads likewise follow its stable ID.
func (k changeKind) item(r integrationDetail, id int) (path string, active bool, err error) {
	base := integrationPath(r.ID)
	if k == changeIntegration && r.ID == id {
		return base, *r.Active, nil
	}
	for _, s := range r.Sources {
		if k == changeSource && s.ID == id {
			return fmt.Sprintf("%s/sources/%d", base, id), *s.Active, nil
		}
		if k == changeGrant {
			for _, g := range s.Grants {
				if g.ID == id {
					return fmt.Sprintf("%s/sources/%d/grants/%d", base, s.ID, id), *g.Active, nil
				}
			}
		}
	}
	return "", false, ErrNotFound
}

func (k changeKind) update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fields := []string{"active"}
	switch k {
	case changeIntegration:
		fields = append(fields, "name")
	case changeSource:
		fields = append(fields, "source", "owner_team_id", "ordering_mode")
	case changeGrant:
		fields = append(fields, "source_id", "capability", "target_team_id", "target_service_id")
	}
	body := map[string]interface{}{}
	for _, field := range fields {
		if d.HasChange(field) {
			v := d.Get(field)
			// Explicit null returns the grant to Team scope.
			if field == "target_service_id" && v.(int) == 0 {
				v = nil
			}
			body[field] = v
		}
	}
	err := k.patch(ctx, d, meta.(*Config).Client, body, false)
	return diag.FromErr(err)
}

func (k changeKind) remove(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := k.patch(ctx, d, meta.(*Config).Client, map[string]interface{}{"active": false}, true)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func (k changeKind) patch(ctx context.Context, d *schema.ResourceData, client *Client, body map[string]interface{}, destroy bool) error {
	client.changeWrite.Lock()
	defer client.changeWrite.Unlock()
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	parent := k.parentID(d, id)
	r, err := changeRequest(ctx, client, http.MethodGet, integrationPath(parent), nil, parent)
	if err != nil {
		return err
	}
	path, active, err := k.item(r, id)
	if err != nil {
		return err
	}
	if destroy && !active {
		return nil
	}
	if len(body) > 0 {
		body["expected_revision"] = r.Revision
		r, err = changeRequest(ctx, client, http.MethodPatch, path, body, parent)
		// Do not replay ambiguous writes or hide a concurrent administrator's 409.
		if err != nil {
			return err
		}
	}
	if destroy {
		_, active, err = k.item(r, id)
		if err != nil {
			return err
		}
		if active {
			return fmt.Errorf("IR did not deactivate Change %s %d", k, id)
		}
		return nil
	}
	return k.save(d, r)
}
