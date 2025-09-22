package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

type OrderItem struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type Order struct {
	Items []OrderItem `json:"items"`
}

// Field tokens for presence-safe gating
var (
	fieldOrderItems = goskema.FieldOf[Order](func(o *Order) *[]OrderItem { return &o.Items })
)

// External services

type Catalog interface {
	SKUExists(ctx context.Context, sku string) (bool, error)
}

type Inventory interface {
	Available(ctx context.Context, sku string) (int, error)
	Reserve(ctx context.Context, sku string, qty int) (bool, error)
	Release(ctx context.Context, sku string, qty int) error
}

// In-memory implementations

type memCatalog struct{ m map[string]struct{} }

func (c memCatalog) SKUExists(ctx context.Context, sku string) (bool, error) {
	_, ok := c.m[sku]
	return ok, nil
}

type memInventory struct {
	mu sync.Mutex
	m  map[string]int
}

func (inv *memInventory) Available(ctx context.Context, sku string) (int, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if n, ok := inv.m[sku]; ok {
		return n, nil
	}
	return 0, nil
}

func (inv *memInventory) Reserve(ctx context.Context, sku string, qty int) (bool, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	n, ok := inv.m[sku]
	if !ok || qty <= 0 {
		return false, nil
	}
	if n < qty {
		return false, nil
	}
	inv.m[sku] = n - qty
	return true, nil
}

func (inv *memInventory) Release(ctx context.Context, sku string, qty int) error {
	if qty <= 0 {
		return nil
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.m[sku] = inv.m[sku] + qty
	return nil
}

func orderSchema() goskema.Schema[Order] {
	item := g.ObjectOf[OrderItem]().
		Field("sku", g.StringOf[string]()).Required().
		Field("qty", g.IntOf[int]().Min(1)).Required().
		UnknownStrict().
		MustBind()

	return g.ObjectOf[Order]().
		Field("items", g.ArrayOfSchema(g.Array(item).Min(1))).Required().
		UnknownStrict().
		// Domain: qty positive is now enforced at wire-level via Min(1)
		// Domain: SKU uniqueness across items
		RefineT("sku_unique", func(dc goskema.DomainCtx[Order], o Order) []goskema.Issue {
			if !dc.AnySeen(fieldOrderItems) {
				return nil
			}
			seen := map[string]int{}
			var out []goskema.Issue
			rItems := dc.Path(fieldOrderItems)
			keySKU := goskema.FieldNameOf(func(oi *OrderItem) *string { return &oi.SKU })
			for i, it := range o.Items {
				if j, ok := seen[it.SKU]; ok {
					pr := rItems.Index(i).Field(keySKU)
					msg := fmt.Sprintf("SKU duplicates items[%d].%s", j, keySKU)
					out = append(out, pr.Issue(goskema.CodeUniqueness, msg, "first", j, "dup", i, keySKU, it.SKU))
				} else {
					seen[it.SKU] = i
				}
			}
			return out
		}).
		// Context: SKU must exist in catalog
		RefineCtxE("sku_exists", func(dc goskema.DomainCtx[Order], o Order) ([]goskema.Issue, error) {
			if !dc.AnySeen(fieldOrderItems) {
				return nil, nil
			}
			cat, err := goskema.RequireService[Catalog](dc.Ctx)
			if err != nil {
				return nil, fmt.Errorf("catalog service unavailable")
			}
			// dedupe SKUs and check once per sku
			idxsBySKU := map[string][]int{}
			for i, it := range o.Items {
				idxsBySKU[it.SKU] = append(idxsBySKU[it.SKU], i)
			}
			skus := make([]string, 0, len(idxsBySKU))
			for s := range idxsBySKU {
				skus = append(skus, s)
			}
			sort.Strings(skus)
			var out []goskema.Issue
			rItems := dc.Path(fieldOrderItems)
			keySKU := goskema.FieldNameOf(func(oi *OrderItem) *string { return &oi.SKU })
			for _, sku := range skus {
				ok, e := cat.SKUExists(dc.Ctx, sku)
				if e != nil {
					return nil, fmt.Errorf("catalog lookup failed: %w", e)
				}
				if !ok {
					for _, i := range idxsBySKU[sku] {
						out = append(out, rItems.Index(i).Field(keySKU).Issue(goskema.CodeBusinessRule, "SKU not found", "sku", sku))
					}
				}
			}
			return out, nil
		}).
		// Context: inventory must be sufficient
		RefineCtxE("inventory_sufficient", func(dc goskema.DomainCtx[Order], o Order) ([]goskema.Issue, error) {
			if !dc.AnySeen(fieldOrderItems) {
				return nil, nil
			}
			inv, err := goskema.RequireService[Inventory](dc.Ctx)
			if err != nil {
				return nil, fmt.Errorf("inventory service unavailable")
			}
			// aggregate demand per SKU to compare against available atomically
			idxsBySKU := map[string][]int{}
			needBySKU := map[string]int{}
			for i, it := range o.Items {
				if it.Qty > 0 { // validated separately by qty_positive
					idxsBySKU[it.SKU] = append(idxsBySKU[it.SKU], i)
					needBySKU[it.SKU] += it.Qty
				}
			}
			skus := make([]string, 0, len(needBySKU))
			for s := range needBySKU {
				skus = append(skus, s)
			}
			sort.Strings(skus)
			var out []goskema.Issue
			rItems := dc.Path(fieldOrderItems)
			keyQty := goskema.FieldNameOf(func(oi *OrderItem) *int { return &oi.Qty })
			for _, sku := range skus {
				need := needBySKU[sku]
				avail, e := inv.Available(dc.Ctx, sku)
				if e != nil {
					return nil, fmt.Errorf("inventory lookup failed for %s: %w", sku, e)
				}
				if avail < need {
					for _, i := range idxsBySKU[sku] {
						out = append(out, rItems.Index(i).Field(keyQty).Issue(
							goskema.CodeAggregateViolation,
							"insufficient inventory",
							"available", avail, "requested", need, "sku", sku,
						))
					}
				}
			}
			return out, nil
		}).
		MustBind()
}

func printIssues(err error) bool {
	if iss, ok := goskema.AsIssues(err); ok {
		b, _ := json.MarshalIndent(map[string]any{"issues": iss}, "", "  ")
		fmt.Println(string(b))
		return true
	}
	return false
}

// allocateOrder attempts to reserve inventory for the order atomically.
// It first verifies aggregated availability, then performs per-SKU reservation.
// On partial failure, it releases any reservations done so far.
func allocateOrder(ctx context.Context, inv Inventory, o Order) error {
	// aggregate
	demand := map[string]int{}
	for _, it := range o.Items {
		if it.Qty <= 0 {
			return fmt.Errorf("invalid qty for sku %s", it.SKU)
		}
		demand[it.SKU] += it.Qty
	}
	// check first
	for sku, need := range demand {
		avail, err := inv.Available(ctx, sku)
		if err != nil {
			return fmt.Errorf("inventory lookup failed for %s: %w", sku, err)
		}
		if avail < need {
			return fmt.Errorf("insufficient inventory for %s: need=%d available=%d", sku, need, avail)
		}
	}
	// reserve
	reserved := map[string]int{}
	for sku, need := range demand {
		ok, err := inv.Reserve(ctx, sku, need)
		if err != nil || !ok {
			// rollback
			for rsku, qty := range reserved {
				_ = inv.Release(ctx, rsku, qty)
			}
			if err != nil {
				return fmt.Errorf("reserve failed for %s: %w", sku, err)
			}
			return fmt.Errorf("reserve failed for %s", sku)
		}
		reserved[sku] = need
	}
	return nil
}

func main() {
	ctx := context.Background()
	cat := memCatalog{m: map[string]struct{}{"SKU-A": {}, "SKU-B": {}}}
	inv := &memInventory{m: map[string]int{"SKU-A": 10, "SKU-B": 0}}
	ctx = goskema.WithService(ctx, Catalog(cat))
	ctx = goskema.WithService(ctx, Inventory(inv))

	s := orderSchema()

	fmt.Println("-- OK: one item, exists, sufficient, then allocate --")
	if v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"items":[{"sku":"SKU-A","qty":2}]}`))); err != nil {
		if !printIssues(err) {
			fmt.Println("error:", err)
		}
	} else {
		if err := allocateOrder(ctx, inv, v); err != nil {
			fmt.Println("allocation error:", err)
		} else {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
			fmt.Println("inventory after allocation:", inv.m)
		}
	}

	fmt.Println("\n-- NG: empty items --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"items":[]}`))); err != nil {
		if !printIssues(err) {
			fmt.Println("error:", err)
		}
	}

	fmt.Println("\n-- NG: duplicate SKU --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"items":[{"sku":"SKU-A","qty":1},{"sku":"SKU-A","qty":3}]}`))); err != nil {
		if !printIssues(err) {
			fmt.Println("error:", err)
		}
	}

	fmt.Println("\n-- NG: SKU not found --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"items":[{"sku":"NOPE","qty":1}]}`))); err != nil {
		if !printIssues(err) {
			fmt.Println("error:", err)
		}
	}

	fmt.Println("\n-- NG: insufficient inventory (no allocation) --")
	if v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"items":[{"sku":"SKU-B","qty":1}]}`))); err != nil {
		if !printIssues(err) {
			fmt.Println("error:", err)
		}
	} else {
		if err := allocateOrder(ctx, inv, v); err != nil {
			fmt.Println("allocation error:", err)
		}
	}
}
