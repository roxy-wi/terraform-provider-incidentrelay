package incidentrelay

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func sampleSystemSettings() map[string]interface{} {
	// Nonzero values and enabled switches make unintentional resets observable.
	return map[string]interface{}{
		"change_intake_enabled": true, "change_worker_enabled": true,
		"change_max_future_seconds": 300, "change_events_per_minute": 600, "change_events_per_change": 1000,
		"change_worker_interval_seconds": 5, "change_worker_jobs_per_tick": 20, "change_worker_batch_size": 100,
		"change_worker_lease_seconds": 60, "change_worker_max_attempts": 8, "change_worker_retry_base_seconds": 5,
		"change_worker_retry_max_seconds": 300, "change_pending_ttl_seconds": 3600, "change_pending_poll_seconds": 10,
	}
}

func TestSystemSettingsPartialPatch(t *testing.T) {
	ctx := context.Background()
	remote := sampleSystemSettings()
	var payloads []map[string]interface{}
	c := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != systemSettingsPath {
			t.Error(r.URL.Path)
		}
		switch r.Method {
		case http.MethodPatch:
			var p map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			payloads = append(payloads, p)
			for k, v := range p {
				remote[k] = v
			}
		case http.MethodGet:
		default:
			t.Errorf("unexpected %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(remote); err != nil {
			t.Error(err)
		}
	})
	r := resourceSystemSettings()
	var state *terraform.InstanceState
	apply := func(attrs map[string]string) {
		t.Helper()
		block := r.CoreConfigSchema()
		raw, err := (&terraform.InstanceState{Attributes: attrs}).AttrsAsObjectValue(block.ImpliedType())
		if err != nil {
			t.Fatal(err)
		}
		cfg := terraform.NewResourceConfigShimmed(raw, block)
		cfg.CtyValue = raw
		diff, err := r.SimpleDiff(ctx, state, cfg, c)
		if err != nil {
			t.Fatal(err)
		}
		diff.RawConfig = raw
		next, diags := r.Apply(ctx, state, diff, c)
		if diags.HasError() {
			t.Fatal(diags)
		}
		state = next
	}
	apply(map[string]string{"change_intake_enabled": "false", "change_max_future_seconds": "0"})
	apply(map[string]string{"change_intake_enabled": "true", "change_worker_enabled": "true"})
	want := []map[string]interface{}{{"change_intake_enabled": false, "change_max_future_seconds": float64(0)}, {"change_intake_enabled": true}}
	if state.ID != systemSettingsID || !reflect.DeepEqual(payloads, want) {
		t.Fatalf("id=%s patches=%#v", state.ID, payloads)
	}
}

func TestSystemSettingsErrorsRetainBinding(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"jwt", 401, "{}"}, {"role", 403, "{}"}, {"old-server", 404, "{}"},
		{"incomplete", 200, `{"change_intake_enabled":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			d := schema.TestResourceDataRaw(t, resourceSystemSettings().Schema, nil)
			d.SetId(systemSettingsID)
			if !readSystemSettings(context.Background(), d, c).HasError() || d.Id() != systemSettingsID {
				t.Fatal("failed read lost binding or succeeded")
			}
		})
	}
}

func TestSystemSettingsBoundsAndIdentity(t *testing.T) {
	r := resourceSystemSettings()
	for _, tc := range []struct {
		name     string
		min, max int
	}{
		{"change_max_future_seconds", 0, 2147483647}, {"change_events_per_change", 1, 10000}, {"change_worker_batch_size", 1, 1000},
		{"change_worker_jobs_per_tick", 1, 100}, {"change_worker_lease_seconds", 10, 2147483647}, {"change_pending_ttl_seconds", 60, 2147483647}, {"change_pending_poll_seconds", 5, 2147483647},
	} {
		check := r.Schema[tc.name].ValidateFunc
		for _, n := range []int{tc.min, tc.max} {
			if _, errs := check(n, tc.name); len(errs) > 0 {
				t.Error(errs)
			}
		}
		if _, errs := check(tc.min-1, tc.name); len(errs) == 0 {
			t.Errorf("%s accepted value below minimum", tc.name)
		}
		if tc.max < 2147483647 {
			if _, errs := check(tc.max+1, tc.name); len(errs) == 0 {
				t.Errorf("%s accepted value above maximum", tc.name)
			}
		}
	}
	d := schema.TestResourceDataRaw(t, r.Schema, nil)
	d.SetId("some-group")
	if _, err := r.Importer.StateContext(context.Background(), d, nil); err == nil {
		t.Fatal("accepted a non-singleton import")
	}
}
