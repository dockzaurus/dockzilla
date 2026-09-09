package images

import (
	"context"
	"strings"
	"testing"
)

func TestFakeStore_LoadAndResolve(t *testing.T) {
    ctx := context.Background()
    store := &FakeStore{
        digestToID: make(map[string]ImageID),
        images:     make(map[ImageID]Image),
    }

    // 1. Test Load
    data := "fake-tarball-content"
    r := strings.NewReader(data)

    digest, err := store.Load(ctx, r)
    if err != nil {
        t.Fatalf("unexpected error on Load: %v", err)
    }

    if !strings.HasPrefix(digest, "sha256:") {
        t.Errorf("expected digest to start with 'sha256:', got %q", digest)
    }

    // 2. Test Resolve
    id, found, err := store.Resolve(ctx, digest)
    if err != nil {
        t.Fatalf("unexpected error on Resolve: %v", err)
    }
    if !found {
        t.Errorf("expected image to be found")
    }
    if id == (ImageID{}) {
        t.Errorf("expected non-empty ImageID")
    }
}

func TestFakeStore_InventoryAndRemove(t *testing.T) {
    ctx := context.Background()
    store := &FakeStore{
        digestToID: make(map[string]ImageID),
        images:     make(map[ImageID]Image),
    }

    r := strings.NewReader("some-image-data")
    digest, _ := store.Load(ctx, r)
    id, _, _ := store.Resolve(ctx, digest)

    // 3. Test Inventory
    inv, err := store.Inventory(ctx)
    if err != nil {
        t.Fatalf("unexpected error on Inventory: %v", err)
    }
    if len(inv) != 1 {
        t.Fatalf("expected 1 image in inventory, got %d", len(inv))
    }
    if inv[0].Digest != digest {
        t.Errorf("expected digest %q, got %q", digest, inv[0].Digest)
    }

    // 4. Test Remove
    reclaimed, err := store.Remove(ctx, id)
    if err != nil {
        t.Fatalf("unexpected error on Remove: %v", err)
    }
    if reclaimed <= 0 {
        t.Errorf("expected positive reclaimed bytes, got %d", reclaimed)
    }

    // Verify it is completely gone
    invAfter, _ := store.Inventory(ctx)
    if len(invAfter) != 0 {
        t.Errorf("expected empty inventory after remove, got %d", len(invAfter))
    }

    _, found, _ := store.Resolve(ctx, digest)
    if found {
        t.Errorf("expected image to not be found after removal")
    }
}