package app

import (
	"context"
	"os"
	"testing"
)

// requireLiveCreds skips the test unless WEWORK_USERNAME and WEWORK_PASSWORD are
// set, keeping the default test run hermetic. The Service reads the same env vars
// the deployed server uses, so these exercise the real tool path end to end.
func requireLiveCreds(t *testing.T) {
	t.Helper()
	if os.Getenv("WEWORK_USERNAME") == "" || os.Getenv("WEWORK_PASSWORD") == "" {
		t.Skip("set WEWORK_USERNAME and WEWORK_PASSWORD to run live service tests")
	}
}

func TestLiveFavorites(t *testing.T) {
	requireLiveCreds(t)
	svc := NewService()
	res, err := svc.Favorites(context.Background(), FavoritesInput{SpaceType: 0})
	if err != nil {
		t.Fatalf("Favorites: %v", err)
	}
	t.Logf("favorites=%d recents=%d", len(res.Items), len(res.Recents))
	for _, f := range res.Items {
		t.Logf("favorite: %s (%s)", f.LocationName, f.LocationID)
	}
}

// TestLiveAddRemoveFavorite exercises the write path against the real account. It
// finds a location that is NOT currently favorited, favorites it, verifies it
// appears, then removes it and verifies it is gone — leaving the account in its
// original state. A mid-run failure leaves at most one extra favorite rather than
// dropping an existing one.
func TestLiveAddRemoveFavorite(t *testing.T) {
	requireLiveCreds(t)
	ctx := context.Background()
	svc := NewService()

	const spaceType = 0

	before, err := svc.Favorites(ctx, FavoritesInput{SpaceType: spaceType})
	if err != nil {
		t.Fatalf("Favorites (before): %v", err)
	}
	favored := map[string]bool{}
	for _, f := range before.Items {
		favored[f.LocationID] = true
	}

	locs, err := svc.Locations(ctx, LocationsInput{City: "Tokyo"})
	if err != nil {
		t.Fatalf("Locations: %v", err)
	}
	var targetUUID, targetName string
	for _, l := range locs.Items {
		if !favored[l.UUID] {
			targetUUID, targetName = l.UUID, l.Name
			break
		}
	}
	if targetUUID == "" {
		t.Skip("no non-favorited Tokyo location available to test with")
	}
	t.Logf("target: %s (%s)", targetName, targetUUID)

	present := func() bool {
		res, err := svc.Favorites(ctx, FavoritesInput{SpaceType: spaceType})
		if err != nil {
			t.Fatalf("Favorites (check): %v", err)
		}
		for _, f := range res.Items {
			if f.LocationID == targetUUID {
				return true
			}
		}
		return false
	}

	input := FavoriteMutationInput{LocationUUID: targetUUID, SpaceType: spaceType}

	if _, err := svc.AddFavorite(ctx, input); err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}
	if !present() {
		t.Errorf("favorite not present after AddFavorite")
	} else {
		t.Log("added OK")
	}

	if _, err := svc.RemoveFavorite(ctx, input); err != nil {
		t.Fatalf("RemoveFavorite: %v", err)
	}
	if present() {
		t.Errorf("favorite still present after RemoveFavorite")
	} else {
		t.Log("removed OK; account restored")
	}
}

func TestLivePrintQueue(t *testing.T) {
	requireLiveCreds(t)
	svc := NewService()
	res, err := svc.PrintQueue(context.Background(), PrintQueueInput{})
	if err != nil {
		t.Fatalf("PrintQueue: %v", err)
	}
	t.Logf("print jobs=%d totalElements=%d", len(res.Content), res.Page.TotalElements)
}
