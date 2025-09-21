package main

import (
	"context"
	"encoding/json"
	"fmt"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
	r "github.com/reoring/goskema/rules"
)

type OrderItem struct {
	SKU   string `json:"sku"`
	Qty   int    `json:"qty"`
	Price int    `json:"price"`
}

type Order struct {
	Status string      `json:"status"` // "QUOTE" | "CONFIRMED"
	Items  []OrderItem `json:"items"`
}

func orderSchema() goskema.Schema[Order] {
	item := g.ObjectOf[OrderItem]().
		Field("sku", g.StringOf[string]()).Required().
		Field("qty", g.IntOf[int]()).Required().
		Field("price", g.IntOf[int]()).Required().
		UnknownStrict().
		MustBind()

	// local helper rules to demonstrate And/Or
	requireStatus := func(expect string) r.Rule[Order] {
		return func(dc goskema.DomainCtx[Order], o Order) []goskema.Issue {
			if o.Status == expect {
				return nil
			}
			return []goskema.Issue{dc.Ref.At("/status").Issue(goskema.CodeBusinessRule, fmt.Sprintf("status must be %s or meet alternative rule", expect), "expect", expect)}
		}
	}
	allQtyPositive := func() r.Rule[Order] {
		return func(dc goskema.DomainCtx[Order], o Order) []goskema.Issue {
			var out []goskema.Issue
			for i, it := range o.Items {
				if it.Qty <= 0 {
					out = append(out, dc.Ref.At("/items").Index(i).Field("qty").Issue(goskema.CodeDomainRange, "qty must be > 0", "min", 1, "got", it.Qty))
				}
			}
			return out
		}
	}

	return g.ObjectOf[Order]().
		Field("status", g.StringOf[string]()).Required().
		Field("items", g.ArrayOf(item)).Required().
		UnknownStrict().
		// Or: if status is QUOTE it's allowed; otherwise require items >= 1
		RefineT("items_policy",
			r.Or(
				requireStatus("QUOTE"),
				r.AtLeastOne[Order]("/items"),
			),
		).
		// And: enforce SKU uniqueness and qty > 0
		RefineT("items_integrity",
			r.And(
				r.UniqueBy[Order]("/items", "sku"),
				allQtyPositive(),
			),
		).
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

	fmt.Println("\n-- OK (QUOTE with 0 items: allowed by Or) --")
	if v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"status":"QUOTE","items":[]}`))); err != nil {
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

	fmt.Println("\n-- NG (qty <= 0) --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"status":"CONFIRMED","items":[{"sku":"A","qty":0,"price":100}]}`))); err != nil {
		_ = printIssues(err)
	}
}
