package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func changeTestClient(t *testing.T, handler http.HandlerFunc) *Config {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(ClientConfig{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &Config{Client: c}
}

func TestConcurrentChangeGrantsUseFreshRevision(t *testing.T) {
	active := true
	state := integrationDetail{ID: 21, Active: &active, Revision: 1, Sources: []sourceDetail{{ID: 7, Active: &active, Name: "build", Grants: []grantDetail{}}}}
	var mu sync.Mutex
	cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == "GET" && r.URL.Path == integrationPath(21):
		case r.Method == "POST" && r.URL.Path == integrationPath(21)+"/sources/7/grants":
			var body struct {
				Revision   int64  `json:"expected_revision"`
				Team       int    `json:"target_team_id"`
				Capability string `json:"capability"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if body.Revision != state.Revision {
				w.WriteHeader(409)
				return
			}
			state.Revision++
			// New item first: callers must use its identity, not the last array item.
			state.Sources[0].Grants = append([]grantDetail{{ID: 100 + body.Team, Active: &active, Team: body.Team, Capability: body.Capability}}, state.Sources[0].Grants...)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(405)
			return
		}
		if err := json.NewEncoder(w).Encode(state); err != nil {
			t.Error(err)
		}
	})
	start := make(chan struct{})
	var wg sync.WaitGroup
	for team := 1; team <= 6; team++ {
		d := schema.TestResourceDataRaw(t, changeResource(changeGrant).Schema, map[string]interface{}{"integration_id": 21, "source_id": 7, "target_team_id": team, "capability": "changed"})
		wg.Add(1)
		go func(team int) {
			defer wg.Done()
			<-start
			if diags := changeGrant.create(context.Background(), d, cfg); diags.HasError() {
				t.Errorf("create: %v", diags)
				return
			}
			if d.Id() != strconv.Itoa(100+team) {
				t.Errorf("incorrect grant ID %s for Team %d", d.Id(), team)
			}
		}(team)
	}
	close(start)
	wg.Wait()
}

func TestChangeFailedDestroyRetainsState(t *testing.T) {
	for _, code := range []int{401, 403, 409, 500} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			var attempts atomic.Int32
			cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					fmt.Fprint(w, `{"id":4,"active":true,"auth_revision":3,"sources":[]}`)
					return
				}
				if r.Method != "PATCH" {
					t.Errorf("unexpected %s", r.Method)
				}
				attempts.Add(1)
				w.WriteHeader(code)
			})
			d := schema.TestResourceDataRaw(t, changeResource(changeIntegration).Schema, nil)
			d.SetId("4")
			if diags := changeIntegration.remove(context.Background(), d, cfg); !diags.HasError() || d.Id() != "4" || attempts.Load() != 1 {
				t.Fatalf("failed write lost identity or was replayed: %v, id=%s, attempts=%d", diags, d.Id(), attempts.Load())
			}
		})
	}
}

func TestChangeRefreshAndDestroyMissing(t *testing.T) {
	for _, status := range []int{200, 403, 404} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			calls := atomic.Int32{}
			cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				if status == 200 {
					fmt.Fprint(w, `{"id":7,"active":true,"auth_revision":3,"sources":[]}`)
				}
			})
			for _, op := range []func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics{changeSource.read, changeSource.remove} {
				d := schema.TestResourceDataRaw(t, changeResource(changeSource).Schema, map[string]interface{}{"integration_id": 7})
				d.SetId("12")
				ds := op(context.Background(), d, cfg)
				if ds.HasError() != (status == 403) || (d.Id() == "") != (status != 403) {
					t.Fatalf("unexpected state or diagnostic: id %q, %v", d.Id(), ds)
				}
			}
			if calls.Load() != 2 {
				t.Fatal("unexpected writes for missing resource")
			}
		})
	}
}

func TestChangeImportChecksParentsAndIDs(t *testing.T) {
	var calls atomic.Int32
	cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"id":9,"active":true,"auth_revision":7,"sources":[{"id":2,"active":true,"grants":[{"id":3,"active":false}]}]}`)
	})
	for _, id := range []string{"3", "0/2/3", "9/2/abc", "9/-1/3", "9/4/3", "9/2/999"} {
		d := schema.TestResourceDataRaw(t, changeResource(changeGrant).Schema, nil)
		d.SetId(id)
		if _, err := changeGrant.importState(context.Background(), d, cfg); err == nil {
			t.Errorf("invalid import %s succeeded", id)
		}
	}
	if calls.Load() != 2 {
		t.Errorf("malformed IDs should fail before HTTP; got %d reads", calls.Load())
	}
}

func TestInactiveChangeCreateDoesNotGrantAuthority(t *testing.T) {
	cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected %s %s", r.Method, r.URL.Path) })
	for _, kind := range []changeKind{changeGrant, changeSource, changeIntegration} {
		d := schema.TestResourceDataRaw(t, changeResource(kind).Schema, map[string]interface{}{"active": false})
		if !kind.create(context.Background(), d, cfg).HasError() {
			t.Errorf("create accepted inactive %s", kind)
		}
	}
}

func TestChangeIncompleteDetailKeepsGrantInState(t *testing.T) {
	for _, body := range []string{
		`{"id":9,"active":true,"auth_revision":4,"sources":[{"id":7,"active":true}]}`,
		`{"id":9,"active":true,"auth_revision":4,"sources":[{"id":7,"active":true,"grants":[{"id":2}]}]}`,
	} {
		cfg := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		d := schema.TestResourceDataRaw(t, changeResource(changeGrant).Schema, map[string]interface{}{"integration_id": 9, "source_id": 7})
		d.SetId("2")
		if !changeGrant.read(context.Background(), d, cfg).HasError() || d.Id() != "2" {
			t.Fatal("incomplete response was treated as missing/inactive grant")
		}
	}
}

func TestCorrelationExclusionValidation(t *testing.T) {
	valid := []string{`[]`, `[{"source":"heartbeat"}]`, `[{"labels":{"environment":"test","empty":""}}]`, `[{"source":null,"labels":{"a":"b"}}]`}
	invalid := []string{"null", "{}", `[null]`, `[{}]`, `[{"labels":{}}]`, `[{"source":1}]`, `[{"labels":{"x":null}}]`, `[{"labels":{"":"x"}}]`, `[{"unexpected":{}}]`, `[{"source":" trailing "}]`, `[{"source":"` + strings.Repeat("а", 256) + `"}]`, `[` + strings.Repeat(`{"source":"s"},`, 50) + `{"source":"s"}]`}
	validate := resourceService().Schema["change_correlation_exclusions_json"].ValidateFunc
	for _, v := range valid {
		if _, errs := validate(v, "rules"); len(errs) != 0 {
			t.Errorf("rejected %s: %v", v, errs)
		}
	}
	for _, v := range invalid {
		if _, errs := validate(v, "rules"); len(errs) == 0 {
			t.Errorf("accepted %s", v)
		}
	}
}
