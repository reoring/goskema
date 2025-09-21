package main

import (
	"context"
	"encoding/json"
	"fmt"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

type UserRef struct {
	UserID string `json:"userId"`
}

type Order struct {
	User   UserRef `json:"user"`
	Total  int     `json:"total"`
	Status string  `json:"status"`
}

// Typed field tokens for presence-safe gating
var (
	fieldOrderUser   = goskema.FieldOf[Order](func(o *Order) *UserRef { return &o.User })
	fieldOrderStatus = goskema.FieldOf[Order](func(o *Order) *string { return &o.Status })
	pathUserUserID   = goskema.PathOf[Order, string](func(o *Order) *string { return &o.User.UserID })
)

// UserStore represents an external dependency to check user existence.
type UserStore interface {
	Exists(ctx context.Context, id string) (bool, error)
}

type memUserStore struct{ m map[string]struct{} }

func (s memUserStore) Exists(ctx context.Context, id string) (bool, error) {
	_, ok := s.m[id]
	return ok, nil
}

func orderSchema(us UserStore) goskema.Schema[Order] {
	userRef := g.ObjectOf[UserRef]().
		Field("userId", g.StringOf[string]()).Required().
		UnknownStrict().
		MustBind()

	return g.ObjectOf[Order]().
		Field("user", g.SchemaOf(userRef)).Required().
		Field("total", g.IntOf[int]()).Required().
		Field("status", g.StringOf[string]()).Required().
		UnknownStrict().
		// Context rule: user must exist in UserStore when provided
		RefineCtxE("user_exists", func(dc goskema.DomainCtx[Order], o Order) ([]goskema.Issue, error) {
			// run only when user or status are present (avoid over-validation on PATCH)
			if !dc.AnySeen(fieldOrderUser, fieldOrderStatus) {
				return nil, nil
			}
			store, err := goskema.RequireService[UserStore](dc.Ctx)
			if err != nil {
				return nil, fmt.Errorf("user store unavailable")
			}
			ok, e := store.Exists(dc.Ctx, o.User.UserID)
			if e != nil {
				return nil, fmt.Errorf("user lookup failed: %w", e)
			}
			if !ok {
				return []goskema.Issue{dc.PathTo(pathUserUserID).Issue(goskema.CodeBusinessRule, "user does not exist", "userId", o.User.UserID)}, nil
			}
			return nil, nil
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
	store := memUserStore{m: map[string]struct{}{"u_1": {}}}
	ctx = goskema.WithService(ctx, UserStore(store))

	s := orderSchema(store)

	fmt.Println("-- OK: existing user --")
	if v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"user":{"userId":"u_1"},"total":100,"status":"CONFIRMED"}`))); err != nil {
		_ = printIssues(err)
	} else {
		b, _ := json.Marshal(v)
		fmt.Println(string(b))
	}

	fmt.Println("\n-- NG: missing user --")
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"user":{"userId":"nope"},"total":50,"status":"CONFIRMED"}`))); err != nil {
		_ = printIssues(err)
	}
}
