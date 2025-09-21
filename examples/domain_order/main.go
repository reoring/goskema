package main

import (
	"context"
	"encoding/json"
	"fmt"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

type OrderItem struct {
	SKU   string `json:"sku"`
	Qty   int    `json:"qty"`
	Price int    `json:"price"`
}

type Order struct {
	Status string      `json:"status"` // "QUOTE" | "CONFIRMED" | "CANCELLED"
	Items  []OrderItem `json:"items"`
}

// Precompute typed field tokens for presence-safe checks.
var (
	fieldOrderStatus = goskema.FieldOf[Order](func(o *Order) *string { return &o.Status })
	fieldOrderItems  = goskema.FieldOf[Order](func(o *Order) *[]OrderItem { return &o.Items })
)

func orderSchema() goskema.Schema[Order] {
	item := g.ObjectOf[OrderItem]().
		Field("sku", g.StringOf[string]()).Required().
		Field("qty", g.IntOf[int]()).Required().
		Field("price", g.IntOf[int]()).Required().
		UnknownStrict().
		MustBind()

	return g.ObjectOf[Order]().
		Field("status", g.StringOf[string]()).Required().
		Field("items", g.ArrayOf(item)).Required().
		UnknownStrict().
		// Domain rule: unless QUOTE, require at least one item
		RefineT("items_required_unless_quote", func(dc goskema.DomainCtx[Order], o Order) []goskema.Issue {
			if !dc.AnySeen(fieldOrderStatus, fieldOrderItems) {
				return nil
			}
			if o.Status != "QUOTE" && len(o.Items) == 0 {
				return []goskema.Issue{goskema.NewRef(dc.Presence).At("/items").Issue(goskema.CodeTooShort, "at least 1 item is required", "minItems", 1)}
			}
			return nil
		}).
		// Domain rule: SKU uniqueness across items
		RefineT("sku_unique", func(dc goskema.DomainCtx[Order], o Order) []goskema.Issue {
			seen := map[string]int{}
			var out []goskema.Issue
			for i, it := range o.Items {
				if j, ok := seen[it.SKU]; ok {
					out = append(out, goskema.NewRef(dc.Presence).At("/items").Index(i).Field("sku").Issue(goskema.CodeUniqueness, fmt.Sprintf("SKU duplicates items[%d].sku", j), "first", j, "dup", i, "sku", it.SKU))
				} else {
					seen[it.SKU] = i
				}
			}
			return out
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

func main() {
	ctx := context.Background()
	s := orderSchema()

	fmt.Println("-- OK (CONFIRMED with 1 item) --")
	if v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"status":"CONFIRMED","items":[{"sku":"A","qty":1,"price":100}]}`))); err != nil {
		_ = printIssues(err)
	} else {
		b, _ := json.Marshal(v)
		fmt.Println(string(b))
	}

	fmt.Println("\n-- NG (CONFIRMED with 0 items) --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"status":"CONFIRMED","items":[]}`))); err != nil {
		_ = printIssues(err)
	}

	fmt.Println("\n-- NG (duplicate SKU) --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"status":"CONFIRMED","items":[{"sku":"A","qty":1,"price":100},{"sku":"A","qty":2,"price":150}]}`))); err != nil {
		_ = printIssues(err)
	}
}
