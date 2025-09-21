## goskema ユーザーガイド

高速・型安全なスキーマ/コーデック基盤（Go）である goskema の実用ガイドです。まずはインストールとクイックスタートから始め、主要な機能を順に紹介します。

### インストール
```bash
go get github.com/reoring/goskema
```

### クイックスタート（DSL）
```go
package main

import (
	"context"
	"fmt"
	g "github.com/reoring/goskema/dsl"
	"github.com/reoring/goskema"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func main() {
	// 推奨: 型付きビルダー（DX良好）
  user := g.ObjectOf[User]().
    Field("id",    g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Require("email").
		UnknownStrict().
		MustBind()

	data := []byte(`{"id":"u_1","email":"x@example.com"}`)
	src := goskema.JSONBytes(data)
	u, err := goskema.ParseFrom(context.Background(), user, src)
	fmt.Println(u, err)
}
```

### ストリーミング/制御
巨大入力や制御の強化が必要な場合はストリーミングAPIと `ParseOpt` を利用します。
```go
opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}, MaxDepth: 8, MaxBytes: 1<<20}
_, err := goskema.StreamParse[struct{}](ctx, someSchema, r, opt)
```

#### 巨大配列を逐次イテレートしつつ1件ずつ検証→型投影
```go
// 要素スキーマ（型付き）
type Item struct{ ID string `json:"id"` }
itemS := g.ObjectOf[Item]().
  Field("id", g.StringOf[string]()).Required().
  UnknownStrip().
  MustBind()

// 入力を Source 化
src := goskema.JSONReader(r)
// 配列全体のスキーマ
arr := g.Array[Item](itemS)

// ParseFrom は内部で Source 駆動。Array/Item はサブツリーを使って1件ずつ検証される。
items, err := goskema.ParseFrom(ctx, arr, src, opt)
// 部分成功/失敗を分けたい場合は、`ParseFrom` ではなく Array のストリーミング実装（内部）に倣い、
// 失敗 Issue を収集しつつ成功した Item だけを out に append するパターンが推奨です。
// ライブラリは `Issues` を返すので、必要に応じて `errors.As(err, &goskema.Issues)` で詳細を取得し、
// UI/ログに配列 index つきで可視化できます。
```

### Number の Coerce オプション（既定は非Coerce）
```go
n := g.NumberJSON() // default: string -> number はNG

n2 := g.NumberJSON().CoerceFromString() // 明示オプトイン
v, err := n2.Parse(ctx, "1.0")         // => json.Number("1")
```

### Object ビルダーと UnknownPassthrough（ビルド時検証）
```go
obj, err := g.Object().
  Field("name", g.String()).Required().
  Field("extra", g.MapAny()).
  UnknownPassthrough("extra"). // extra が存在し map[string]any か検証
  Build()
```

### Array × Object（配列要素がオブジェクト）
配列の要素にオブジェクトを入れるには、要素用 `Object().Build()` を作り、それを `Array(...)` に渡します。フィールドに割り当てる場合は `ArrayOf`（制約なし）か、`ArrayOfSchema`（`Min/Max` など制約あり）を使います。

```go
// 1) 要素となるオブジェクト
item, _ := g.Object().
  Field("id",   g.String()).Required().
  Field("name", g.String()).Required().
  UnknownStrict().
  Build()

// 2-a) 制約なしの配列をフィールドに割り当て
listA, _ := g.Object().
  Field("items", g.ArrayOf[map[string]any](item)). // []map[string]any
  UnknownStrict().
  Build()

// 2-b) Min/Max など制約あり配列をフィールドに割り当て
ab := g.Array(item).Min(1).Max(100) // ArrayBuilder[map[string]any]
listB, _ := g.Object().
  Field("items", g.ArrayOfSchema[map[string]any](ab)).
  UnknownStrict().
  Build()

// 3) 使い方
ctx := context.Background()
in := map[string]any{
  "items": []any{
    map[string]any{"id": "1", "name": "A"},
    map[string]any{"id": "2", "name": "B"},
  },
}
v, err := listB.Parse(ctx, in)
_ = err
_ = v["items"].([]map[string]any) // 型: []map[string]any
```

判別共用体（Discriminated Union）を要素にすることもできます。

```go
// バリアントとなるオブジェクト
card, _ := g.Object().
  Field("type", g.String()).
  Field("number", g.String()).Require("number").
  UnknownStrict().
  Build()

bank, _ := g.Object().
  Field("type", g.String()).
  Field("iban", g.String()).Require("iban").
  UnknownStrict().
  Build()

pay := g.Object().
  Discriminator("type").
  OneOf(
    g.Variant("card", card),
    g.Variant("bank", bank),
  ).
  MustBuild() // Schema[map[string]any]

// 配列（少なくとも1件）をフィールドに割り当て
paymentsAB := g.Array(pay).Min(1)
checkout, _ := g.Object().
  Field("payments", g.ArrayOfSchema[map[string]any](paymentsAB)).
  UnknownStrict().
  Build()
```

### Codec（例: RFC3339 <-> time.Time）
```go
import "github.com/reoring/goskema/codec"

c := codec.TimeRFC3339()
// t, _ := c.Decode(ctx, "2025-01-01T00:00:00Z")
```

### Codec のフィールド利用（オブジェクト内）
```go
package main

import (
    "context"
    "time"
    g "github.com/reoring/goskema/dsl"
    "github.com/reoring/goskema"
    "github.com/reoring/goskema/codec"
)

func main() {
    // startDate は wire:string を受け取り domain:time.Time を返す
    obj, _ := g.Object().
        Field("startDate", g.SchemaOf[time.Time](g.Codec[string, time.Time](codec.TimeRFC3339()))).
        Field("title", g.String()).
        Require("startDate", "title").
        UnknownStrict().
        Build()

    js := []byte(`{"startDate":"2024-06-01T00:00:00Z","title":"ok"}`)
    v, err := goskema.ParseFrom(context.Background(), obj, goskema.JSONBytes(js))
    _ = err
    _ = v["startDate"].(time.Time) // time.Time として取り出せる
}
```

```go
// 型付きバインドでキャスト不要にする例
type Event struct {
    StartDate time.Time `json:"startDate"`
    Title     string    `json:"title"`
}

b := g.Object().
  Field("startDate", g.SchemaOf[time.Time](g.Codec[string, time.Time](codec.TimeRFC3339()))).
  Field("title", g.String()).
  Require("startDate", "title").
  UnknownStrict()

eventS, _ := g.Bind[Event](b)
js := []byte(`{"startDate":"2024-06-01T00:00:00Z","title":"ok"}`)
ev, err := goskema.ParseFrom(context.Background(), eventS, goskema.JSONBytes(js))
_ = err
_ = ev.StartDate // そのまま time.Time
```

### 型へのバインド（Bind）
```go
// 関数版（現行Goの制約に親和）
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Alias string `goskema:"name=nickname"`
}

b := g.Object().
  Field("id", g.String()).Required().
  Field("name", g.String()).Required().
  Field("nickname", g.String()).Optional().
  UnknownStrict()

userS, _ := g.Bind[User](b)
// userS.MustParse(...)
```

```go
// チェーン版（型付きビルダー）
userS := g.ObjectOf[User]().
  Field("id", g.StringOf[string]()).
  Field("name", g.StringOf[string]()).
  Field("nickname", g.StringOf[string]()).
  Require("id", "name").
  UnknownStrict().
  MustBind()
```

### どちらをいつ使うか

- **ObjectOf[T] / ObjectTyped[T]（推奨）**
  - フィールド設計と型確定を同じ場で進めたい（DX 重視）
  - 単一のドメイン型に素直に投影するユースケースが中心
  - Compiled 経路とも互換（内部で同じ IR を生成）

  例:
  ```go
  type Profile struct {
      ID    string `json:"id"`
      Name  string `json:"name"`
      Active bool   `json:"active"`
  }

  pS := g.ObjectOf[Profile]().
    Field("id",     g.StringOf[string]()).Required().
    Field("name",   g.StringOf[string]()).Require("name").
    Field("active", g.BoolOf[bool]()).Default(true).
    UnknownStrict().
    MustBind()

  p, err := goskema.ParseFrom(ctx, pS, goskema.JSONBytes([]byte(`{"id":"p1","name":"Reo"}`)))
  _ = err
  _ = p.Active // => true（default）
  ```

- **Object() + Bind[T]**
  - 同一の wire スキーマを複数の `T` に後段で再利用したい
  - 共有ライブラリ/SDK でスキーマだけ配り、アプリ側で型付けしたい
  - 先に JSON Schema / OpenAPI / CRD 出力やコード生成を回し、必要な箇所でだけ型に結合したい
  - テストで「スキーマ妥当性」と「型投影」を段階的に分けて検証したい

  例:
  ```go
  // ライブラリ側（wireスキーマを配布）
  wire := g.Object().
    Field("id", g.String()).Required().
    Field("name", g.String()).Required().
    Field("extra", g.MapAny()).
    UnknownPassthrough("extra").
    MustBuild()

  // アプリ側（用途別に型へ投影）
  type Public struct{ ID, Name string }
  type Admin struct{ ID, Name string; Extra map[string]any `json:"extra"` }

  publicS, _ := g.Bind[Public](wire)
  adminS,  _ := g.Bind[Admin](wire)
  ```

- **補足**
  - `Default/Unknown/Refine` の意味論は両方式で同一
  - 実行時の投影コストは現状わずか（反射）。Compiled 経路ではゼロコスト化可能
  - チーム規約例: ライブラリ層は `Object()+Bind`、アプリ層は `ObjectOf`（`ObjectTyped` 互換）

### Presence（WithMeta の既定は収集オン）
```go
// WithMeta 系（ParseFromWithMeta/DecodeWithMeta）は既定で Presence を収集します。
dm, err := goskema.ParseFromWithMeta(ctx, schema, src)
_ = dm.Presence // nil ではない（Include/Exclude/Intern は必要に応じて指定）

// 収集を明示的に無効化したい場合
dm2, err := goskema.ParseFromWithMeta(ctx, schema, src, goskema.ParseOpt{
    Presence: goskema.PresenceOpt{Collect: false},
})
```

備考:
- Bind のキー解決規則は「DSL明示 > `goskema:"name=..."` > `json`タグ名 > フィールド名」。

### テスト
```bash
go test ./... -race
```

### 関連ドキュメント
- チュートリアル: `docs/tutorial.md`
- DSL リファレンス: `docs/dsl.md`
- リファレンス: `docs/reference.md`
- エラーモデル: `docs/error-model.md`

## NumberMode: 数値の取り扱いを選ぶ

デフォルトでは JSON の数値は `json.Number` として保持します（精度重視）。巨大整数の精度を保ちたい場合に有効です。高速化やライブラリ都合で `float64` に丸めて扱いたい場合は `WithNumberMode` で上書きできます。

```go
ctx := context.Background()
s, _ := g.Object().
  Field("n", g.SchemaOf[json.Number](g.NumberJSON())).
  Require("n").
  UnknownStrict().
  Build()

// 既定（JSONNumber）: 精度保持
v1, _ := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"n":9007199254740993}`)))
_ = v1["n"].(json.Number) // "9007199254740993"

// Float64 モード: 丸めを許容（速度優先）
src := goskema.WithNumberMode(goskema.JSONBytes([]byte(`{"n":9007199254740993}`)), goskema.NumberFloat64)
v2, _ := goskema.ParseFrom(ctx, s, src)
_ = v2["n"].(json.Number) // 内部は float64 を経由した値に規格化
```

用途に応じて NumberMode を選択してください。


