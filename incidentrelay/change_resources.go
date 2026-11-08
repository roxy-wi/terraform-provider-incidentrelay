package incidentrelay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const changeAdminPath = "/api/admin/change-integrations"

type changeKind string

const (
	changeIntegration changeKind = "integration"
	changeSource      changeKind = "source"
	changeGrant       changeKind = "grant"
)

func changeResource(k changeKind) *schema.Resource {
	fields := []fieldDef{optBoolDefault("active", true, "Whether this "+string(k)+" is active. Destroy deactivates it.")}
	switch k {
	case changeIntegration:
		group := optInt("owner_group_id", "Immutable owner Group; omit for a central integration.")
		group.ForceNew = true
		fields = append(fields, reqString("name", "Integration name."), group,
			reqIntForceNew("owner_team_id", "Team for the source created automatically with the integration."),
			computedInt("initial_source_id", "ID of the first source automatically created by IR."),
			computedString("uid", "Stable integration UUID."))
	case changeSource:
		fields = append(fields, reqIntForceNew("integration_id", "Parent integration ID."),
			reqInt("owner_team_id", "Owning Team."), reqString("source", "Producer namespace."),
			optStringDefault("ordering_mode", "timestamp", "timestamp or sequence."),
			computedBool("identity_editable", "Whether IR still allows changing the source name or Team."))
	case changeGrant:
		fields = append(fields, reqIntForceNew("integration_id", "Parent integration ID."), reqInt("source_id", "Source receiving this grant."),
			reqString("capability", "reference, changed or reported_affected."),
			reqInt("target_team_id", "Team to which this permission applies."), optInt("target_service_id", "Limit the grant to one Service; omit to grant the entire Team."))
	}
	s := schemaFromFields(fields)
	for _, name := range []string{"integration_id", "owner_group_id", "owner_team_id", "target_team_id", "target_service_id", "source_id"} {
		if attr := s[name]; attr != nil {
			attr.ValidateFunc = validation.IntAtLeast(1)
		}
	}
	for _, name := range []string{"name", "source"} {
		if attr := s[name]; attr != nil {
			attr.ValidateFunc = func(v interface{}, key string) ([]string, []error) {
				if !validTrimmedText(v.(string), 1, 255) {
					return nil, []error{fmt.Errorf("%s must be 1–255 characters with no surrounding whitespace", key)}
				}
				return nil, nil
			}
		}
	}
	if k == changeSource {
		s["ordering_mode"].ValidateFunc = validation.StringInSlice([]string{"timestamp", "sequence"}, false)
	}
	if k == changeGrant {
		s["capability"].ValidateFunc = validation.StringInSlice([]string{"reference", "changed", "reported_affected"}, false)
	}
	return &schema.Resource{
		Description:   "Manage an IncidentRelay 2.5 Change " + string(k) + ". Requires an administrator JWT.",
		Schema:        s,
		CreateContext: k.create,
		ReadContext:   k.read,
		UpdateContext: k.update,
		DeleteContext: k.remove,
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			if d.Id() == "" && d.NewValueKnown("active") && !d.Get("active").(bool) {
				return fmt.Errorf("IR creates active Change objects only; import an existing object to manage it as inactive")
			}
			return nil
		},
		Importer: &schema.ResourceImporter{StateContext: k.importState},
	}
}

// All endpoints return a fresh detail document for the parent, including on
// child creation. Decode only the public configuration we manage, never tokens.
type integrationDetail struct {
	ID           int            `json:"id"`
	UID          string         `json:"uid"`
	Name         string         `json:"name"`
	OwnerGroupID int            `json:"owner_group_id"`
	Active       *bool          `json:"active"`
	Revision     int64          `json:"auth_revision"`
	Sources      []sourceDetail `json:"sources"`
}
type sourceDetail struct {
	ID       int           `json:"id"`
	Team     int           `json:"owner_team_id"`
	Name     string        `json:"source"`
	Active   *bool         `json:"active"`
	Ordering string        `json:"ordering_mode"`
	Editable bool          `json:"identity_editable"`
	Grants   []grantDetail `json:"grants"`
}
type grantDetail struct {
	ID         int    `json:"id"`
	Active     *bool  `json:"active"`
	Capability string `json:"capability"`
	Team       int    `json:"target_team_id"`
	Service    int    `json:"target_service_id"`
}

func integrationPath(id int) string { return fmt.Sprintf("%s/%d", changeAdminPath, id) }
func changeRequest(ctx context.Context, client *Client, method, path string, body interface{}, expectedID int) (integrationDetail, error) {
	var result integrationDetail
	err := client.Do(ctx, method, path, body, &result)
	if err == nil && !result.complete(expectedID) {
		err = fmt.Errorf("%s %s returned incomplete or unexpected integration detail; refresh before retrying", method, path)
	}
	return result, err
}

// Missing activity or child lists must not look like successful deactivation
// or remote deletion. Keep Terraform's binding on an incomplete response.
func (r integrationDetail) complete(expectedID int) bool {
	if r.ID < 1 || (expectedID != 0 && r.ID != expectedID) || r.Revision < 1 || r.Active == nil || r.Sources == nil {
		return false
	}
	for _, s := range r.Sources {
		if s.ID < 1 || s.Active == nil || s.Grants == nil {
			return false
		}
		for _, g := range s.Grants {
			if g.ID < 1 || g.Active == nil {
				return false
			}
		}
	}
	return true
}

func (k changeKind) parentID(d *schema.ResourceData, id int) int {
	if k == changeIntegration {
		return id
	}
	return d.Get("integration_id").(int)
}

func (k changeKind) read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id, err := strconv.Atoi(d.Id())
	if err != nil || id < 1 {
		return diag.Errorf("invalid Change %s ID %q", k, d.Id())
	}
	parent := k.parentID(d, id)
	result, err := changeRequest(ctx, meta.(*Config).Client, http.MethodGet, integrationPath(parent), nil, parent)
	if err == nil {
		err = k.save(d, result)
	}
	if errors.Is(err, ErrNotFound) {
		d.SetId("")
		return nil
	}
	return diag.FromErr(err)
}

func (k changeKind) save(d *schema.ResourceData, r integrationDetail) error {
	id, err := strconv.Atoi(d.Id())
	if err != nil || id < 1 {
		return fmt.Errorf("invalid Change %s ID %q", k, d.Id())
	}
	switch k {
	case changeIntegration:
		if id != r.ID {
			return fmt.Errorf("unexpected integration ID %d", r.ID)
		}
		values := map[string]interface{}{"name": r.Name, "owner_group_id": r.OwnerGroupID, "active": *r.Active, "uid": r.UID}
		if len(r.Sources) > 0 {
			// The API lists sources by ascending ID and never physically deletes them.
			values["initial_source_id"] = r.Sources[0].ID
			// This is a create-only input, not mutable integration ownership. Seed it
			// on import but do not replace the integration if the initial source moves.
			if _, ok := d.GetOk("owner_team_id"); !ok {
				values["owner_team_id"] = r.Sources[0].Team
			}
		}
		return setChangeState(d, values)
	case changeSource:
		for _, s := range r.Sources {
			if s.ID == id {
				return setChangeState(d, map[string]interface{}{"owner_team_id": s.Team, "source": s.Name, "ordering_mode": s.Ordering, "active": *s.Active, "identity_editable": s.Editable})
			}
		}
	case changeGrant:
		for _, s := range r.Sources {
			for _, g := range s.Grants {
				if g.ID == id {
					return setChangeState(d, map[string]interface{}{"source_id": s.ID, "capability": g.Capability, "target_team_id": g.Team, "target_service_id": g.Service, "active": *g.Active})
				}
			}
		}
	}
	return ErrNotFound
}

func setChangeState(d *schema.ResourceData, values map[string]interface{}) error {
	for field, v := range values {
		if err := d.Set(field, v); err != nil {
			return fmt.Errorf("set %s: %w", field, err)
		}
	}
	return nil
}

func (k changeKind) importState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	want := 1
	if k == changeSource {
		want = 2
	}
	if k == changeGrant {
		want = 3
	}
	if len(parts) != want {
		return nil, fmt.Errorf("Change %s import requires %d positive numeric ID(s) separated by /", k, want)
	}
	ids := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid ID %q", p)
		}
		ids[i] = n
	}
	if k != changeIntegration {
		if err := d.Set("integration_id", ids[0]); err != nil {
			return nil, err
		}
	}
	d.SetId(strconv.Itoa(ids[len(ids)-1]))
	r, err := changeRequest(ctx, meta.(*Config).Client, http.MethodGet, integrationPath(ids[0]), nil, ids[0])
	if err != nil {
		return nil, err
	}
	if err = k.save(d, r); err != nil {
		return nil, err
	}
	if k == changeGrant && d.Get("source_id").(int) != ids[1] {
		return nil, fmt.Errorf("grant %d does not belong to source %d", ids[2], ids[1])
	}
	return []*schema.ResourceData{d}, nil
}
