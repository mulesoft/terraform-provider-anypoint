package exchange

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

func tagList(vals ...string) types.List {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

func stateTags(t *testing.T, l types.List) []string {
	t.Helper()
	out := make([]string, 0, len(l.Elements()))
	for _, e := range l.Elements() {
		out = append(out, e.(types.String).ValueString())
	}
	return out
}

// Exchange canonicalizes labels to lower case. `tags` is Optional+Computed with
// UseStateForUnknown, so a lower-cased read-back replaces the frozen plan value and the
// apply dies with "Provider produced inconsistent result after apply" — for a config as
// ordinary as tags = ["Terraform"].
func TestMapAssetToState_KeepsConfiguredTagCase(t *testing.T) {
	r := &AssetResource{}
	state := &AssetResourceModel{Tags: tagList("Terraform", "PetStore")}

	// What the platform gives back.
	asset := &exchange.Asset{Labels: []string{"terraform", "petstore"}}

	r.mapAssetToState(state, asset)

	got := stateTags(t, state.Tags)
	want := []string{"Terraform", "PetStore"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags = %v, want %v — the configured casing must survive a lower-cased read-back", got, want)
		}
	}
}

// The case-insensitive match must not lose the ordering guarantee it replaced.
func TestMapAssetToState_ReordersTagsIgnoringCase(t *testing.T) {
	r := &AssetResource{}
	state := &AssetResourceModel{Tags: tagList("Alpha", "Beta", "Gamma")}

	// Labels come back in an unrelated order, and lower-cased.
	asset := &exchange.Asset{Labels: []string{"gamma", "alpha", "beta"}}

	r.mapAssetToState(state, asset)

	got := stateTags(t, state.Tags)
	want := []string{"Alpha", "Beta", "Gamma"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags = %v, want %v", got, want)
		}
	}
}

// A label the configuration never mentioned is still surfaced, as the platform spells it.
func TestMapAssetToState_SurfacesUnconfiguredTags(t *testing.T) {
	r := &AssetResource{}
	state := &AssetResourceModel{Tags: tagList("Terraform")}

	asset := &exchange.Asset{Labels: []string{"terraform", "added-in-ui"}}

	r.mapAssetToState(state, asset)

	got := stateTags(t, state.Tags)
	if len(got) != 2 || got[0] != "Terraform" || got[1] != "added-in-ui" {
		t.Fatalf("tags = %v, want [Terraform added-in-ui]", got)
	}
}

// Lower-case configs are the common case and must be untouched.
func TestMapAssetToState_LowercaseTagsUnchanged(t *testing.T) {
	r := &AssetResource{}
	state := &AssetResourceModel{Tags: tagList("terraform", "demo")}

	r.mapAssetToState(state, &exchange.Asset{Labels: []string{"terraform", "demo"}})

	got := stateTags(t, state.Tags)
	if got[0] != "terraform" || got[1] != "demo" {
		t.Fatalf("tags = %v, want [terraform demo]", got)
	}
}
