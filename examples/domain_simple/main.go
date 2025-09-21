package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Example for UnknownPassthrough: keep surplus keys in the "extra" field
type UserWithExtra struct {
	ID    string         `json:"id"`
	Email string         `json:"email"`
	Extra map[string]any `json:"extra,omitempty"`
}

// Presence demo: nickname has a default value
type UserWithPresence struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname,omitempty"`
}

func userSchema() goskema.Schema[User] {
	return g.ObjectOf[User]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		UnknownStrict().
		MustBind()
}

// Schema using the default unknown policy (Strict) without explicitly calling UnknownStrict
func userSchemaDefaultUnknownPolicy() goskema.Schema[User] {
	return g.ObjectOf[User]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		MustBind()
}

// Schema that allows and drops unknown fields (UnknownStrip)
func userSchemaUnknownStrip() goskema.Schema[User] {
	return g.ObjectOf[User]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		UnknownStrip().
		MustBind()
}

// Schema that allows and retains unknown fields (UnknownPassthrough → stored into "extra")
func userSchemaUnknownPassthrough() goskema.Schema[UserWithExtra] {
	return g.ObjectOf[UserWithExtra]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		Field("extra", g.SchemaOf(g.MapAny())).Optional().
		UnknownPassthrough("extra").
		MustBind()
}

// Schema for presence collection (set default for nickname)
func userSchemaWithPresence() goskema.Schema[UserWithPresence] {
	return g.ObjectOf[UserWithPresence]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		Field("nickname", g.StringOf[string]().Nullable()).Default("anon").
		UnknownStrict().
		MustBind()
}

// Wire-level schema allowing null for presence: make nickname nullable
func userSchemaPresenceNullable() goskema.Schema[UserWithPresence] {
	return g.ObjectOf[UserWithPresence]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		Field("nickname", g.StringOf[string]().Nullable()).
		UnknownStrict().
		MustBind()
}

func presenceFlags(p goskema.Presence) []string {
	var out []string
	if p&goskema.PresenceSeen != 0 {
		out = append(out, "Seen")
	}
	if p&goskema.PresenceWasNull != 0 {
		out = append(out, "WasNull")
	}
	if p&goskema.PresenceDefaultApplied != 0 {
		out = append(out, "DefaultApplied")
	}
	return out
}

func renderPresence(pm goskema.PresenceMap) map[string][]string {
	keys := make([]string, 0, len(pm))
	for k := range pm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string][]string, len(pm))
	for _, k := range keys {
		out[k] = presenceFlags(pm[k])
	}
	return out
}

func printIssues(err error) bool {
	if iss, ok := goskema.AsIssues(err); ok {
		b, _ := json.MarshalIndent(map[string]any{"issues": iss}, "", "  ")
		fmt.Println(string(b))
		return true
	}
	return false
}

func exampleSuccess(ctx context.Context) {
	schema := userSchema()
	user, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com"}`)))
	if err != nil {
		if printIssues(err) {
			return
		}
		panic(err)
	}
	b, _ := json.Marshal(user)
	fmt.Println(string(b)) // {"id": "u_1", "email": "x@example.com"}
}

func exampleMissingRequired(ctx context.Context) {
	schema := userSchema()
	// Missing required field "email"
	_, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1"}`)))
	if err != nil {
		_ = printIssues(err)
		// Output:
		// {
		//   "issues": [
		//     {
		//       "Path": "/email",
		//       "Code": "required",
		//       "Message": "required property missing",
		//       "Hint": "required property missing",
		//       "Cause": null,
		//       "Offset": 0,
		//       "InputFragment": ""
		//     }
		//   ]
		// }
		return
	}
}

func exampleUnknownField(ctx context.Context) {
	schema := userSchema()
	// Unknown field "extra" will be rejected under UnknownStrict
	_, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com","extra":true}`)))
	if err != nil {
		_ = printIssues(err)
		return
	}
	fmt.Println("unexpected: unknown field accepted")
}

func exampleUnknownFieldDefault(ctx context.Context) {
	schema := userSchemaDefaultUnknownPolicy()
	// Without UnknownStrict (default is Strict), it still yields unknown_key
	_, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com","zzz":true}`)))
	if err != nil {
		_ = printIssues(err)
		return
	}
	fmt.Println("unexpected: default policy accepted unknown field")
}

func exampleUnknownAllowedStrip(ctx context.Context) {
	schema := userSchemaUnknownStrip()
	// With UnknownStrip, unknown fields are dropped without error
	user, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com","unknown":123}`)))
	if err != nil {
		if printIssues(err) {
			return
		}
		panic(err)
	}
	b, _ := json.Marshal(user)
	fmt.Println(string(b)) // {"id":"u_1","email":"x@example.com"} (unknown field is not output)
}

func exampleUnknownAllowedPassthrough(ctx context.Context) {
	schema := userSchemaUnknownPassthrough()
	// With UnknownPassthrough, unknown fields are preserved in extra
	user, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com","zzz":true,"n":123}`)))
	if err != nil {
		if printIssues(err) {
			return
		}
		panic(err)
	}
	b, _ := json.Marshal(user)
	fmt.Println(string(b)) // {"id":"u_1","email":"x@example.com","extra":{"n":123,"zzz":true}}
}

func examplePresenceDefaultApplied(ctx context.Context) {
	schema := userSchemaWithPresence()
	// nickname missing → default applied (PresenceDefaultApplied)
	dm, err := goskema.ParseFromWithMeta(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com"}`)))
	if err != nil {
		if printIssues(err) {
			return
		}
		panic(err)
	}
	out := map[string]any{
		"value":    dm.Value,
		"presence": renderPresence(dm.Presence),
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func examplePresenceWasNull(ctx context.Context) {
	schema := userSchemaPresenceNullable()
	// nickname is null → WasNull + Seen (nullable)
	dm, err := goskema.ParseFromWithMeta(ctx, schema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com","nickname":null}`)))
	if err != nil {
		if printIssues(err) {
			return
		}
		panic(err)
	}
	out := map[string]any{
		"value":    dm.Value,
		"presence": renderPresence(dm.Presence),
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func main() {
	ctx := context.Background()
	fmt.Println("=== Success case ===")
	exampleSuccess(ctx)
	fmt.Println()

	fmt.Println("=== Unknown key (default policy: Strict) ===")
	exampleUnknownFieldDefault(ctx)
	fmt.Println()

	fmt.Println("=== Unknown key (UnknownStrict explicit) ===")
	exampleUnknownField(ctx)
	fmt.Println()

	fmt.Println("=== Unknown allowed (UnknownStrip: drop) ===")
	exampleUnknownAllowedStrip(ctx)
	fmt.Println()

	fmt.Println("=== Unknown allowed (UnknownPassthrough: keep) ===")
	exampleUnknownAllowedPassthrough(ctx)
	fmt.Println()

	fmt.Println("=== Missing required case ===")
	exampleMissingRequired(ctx)
	fmt.Println()

	fmt.Println("=== Presence (default applied: missing) ===")
	examplePresenceDefaultApplied(ctx)
	fmt.Println()

	fmt.Println("=== Presence (null input) ===")
	examplePresenceWasNull(ctx)
}
