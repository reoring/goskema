## goskema DSL リファレンス

このドキュメントは、goskema の宣言的 DSL（Domain Specific Language）でスキーマを構築し、型安全にパース/検証/投影するための実用的なガイドです。README では概要のみ、詳細な使い方と設計意図は本ドキュメントに集約します。手早く全体像を掴みたい場合は「`docs/dsl-quickstart.md`（チートシート）」も参照してください。

### 目次
- 基本概念（Schema[T], AnyAdapter, Bind/MustBind）
- オブジェクトスキーマ（Field/Required/Require/Default、Unknown ポリシー、Refine、Union）
- プリミティブと数値（String/Bool/Number、NumberOf、NumberMode）
- 配列（最小/最大、ストリーミング）
- マップ（Map、MapAny）
- Presence と WithMeta（欠落/Null/Default の追跡とエンコード）
- ストリーミング（ParseFrom/StreamParse とフォールバック）
- JSON Schema 生成

---

### 基本概念
- Schema[T]: ワイヤ値をドメイン型 T に投影しつつ検証するコアインターフェース。
- AnyAdapter: オブジェクトビルダーのフィールドに挿す“型付きアダプタ”です（例: `StringOf[string]()`）。
- Bind/MustBind: オブジェクトビルダーを型 T に束縛（バインド）します。

最小の例:
```go
import (
    g "github.com/reoring/goskema/dsl"
    "github.com/reoring/goskema"
)

type User struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}

user := g.ObjectOf[User]().
    Field("id",    g.StringOf[string]()).Required().
    Field("email", g.StringOf[string]()).Required().
    UnknownStrict().
    MustBind()

u, err := goskema.ParseFrom(ctx, user, goskema.JSONBytes(data))
```

---

### オブジェクトスキーマ
オブジェクトはビルダーで宣言します。

```go
// ビルダー生成（型付き）
obj := g.ObjectOf[map[string]any]()

// フィールド定義
obj.Field("name",   g.StringOf[string]()).Required()
obj.Field("active", g.BoolOf[bool]()).Default(true)

// 複数を一括必須化
obj.Require("name", "active")

// 未知キーの扱い
obj.UnknownStrict()                 // エラー
// obj.UnknownStrip()               // 受理して破棄
// obj.UnknownPassthrough("extra") // extra へ集約（後述の注意参照）

schema := obj.MustBind()
```

- Required と Require の使い分け（推奨・互換）:
  - 単一フィールドは `Field(...).Required()` を推奨（簡潔でリネーム安全）。
  - 複数同時は `Require("id", "email")` を使用。
  - 互換のため `Field(...).Require("x")` は残していますが、将来削除予定の Deprecated API です。
- Default:
  - `Default(v)` は該当フィールドが欠落時に補完。Presence に `DefaultApplied` が立ちます。
- Unknown ポリシー:
  - Strict: 未知キーはエラー（`unknown_key`）。
  - Strip: 未知キーは受理しつつ捨てる。
  - Passthrough(target): 未知キーを `target` フィールドへ `map[string]any` として集約。`target` は `MapAny()` 相当で受けられる必要があります。

Refine（オブジェクトレベルの後段検証）:
```go
schema := g.Object().
  Field("a", g.StringOf[string]()).
  Field("b", g.StringOf[string]()).
  Refine("a_before_b", func(ctx context.Context, m map[string]any) error {
    if m["a"].(string) >= m["b"].(string) {
        return goskema.Issues{goskema.Issue{Path: "/", Code: "custom", Message: "a must be < b"}}
    }
    return nil
  }).
  MustBuild()
```

Union（判別可能なユニオン）:
```go
u := g.Object().
  Discriminator("type").
  OneOf(
    g.Variant("A", g.Object().Field("type", g.StringOf[string]()).Default("A").MustBuild()),
    g.Variant("B", g.Object().Field("type", g.StringOf[string]()).Default("B").MustBuild()),
  ).
  MustBuild()
```

---

### プリミティブと数値
- `String()` / `Bool()` はワイヤ型へ直接、`StringOf[T]()` / `BoolOf[T]()` はドメイン型 T（基底が string/bool の別名型）へ投影します。
- 数値は JSON 的には `json.Number` を基本にし、`NumberJSON()` でビルダーを得ます。
- 文字列基底で数値文字列を保持するなら `NumberOf[T ~string]()` を、ネイティブ数値に投影するなら以下のショートハンドを使います。
  - 整数: `IntOf[T ~int]()` / `Int32Of[T ~int32]()` / `Int16Of[T ~int16]()` / `Int8Of[T ~int8]()`
  - 非負整数: `UintOf[T ~uint64]()` / `Uint32Of[T ~uint32]()` / `Uint16Of[T ~uint16]()` / `Uint8Of[T ~uint8]()`
  - 浮動小数: `FloatOf[T ~float64]()`

Number の例:
```go
// json.Number を受ける
n := g.NumberJSON() // .CoerceFromString() で文字列からの強制変換も可

// ドメイン型（例: type Price string）へ投影
type Price string
price := g.NumberOf[Price]()

// ネイティブ数値への投影
age   := g.IntOf[int]()
ratio := g.FloatOf[float64]()
tiny  := g.Uint8Of[uint8]()
```

NumberMode（精度/速度の選択）は Source で切り替えます（詳細は README の NumberMode 節参照）。

---

### 配列
```go
type Item struct { ID string `json:"id"` }
item := g.ObjectOf[Item]().
  Field("id", g.StringOf[string]()).Required().
  UnknownStrict().
  MustBind()

arr := g.Array[Item](item).Min(1)
vals, err := goskema.ParseFrom(ctx, arr, goskema.JSONBytes(data))
```

- `Min(n)`, `Max(n)` をサポート。
- ストリーミングに最適化（要素単位でのエラー収集、`/0`, `/1` のようなパス付与）。

---

### マップ
```go
m := g.Map[int](g.NumberJSON())            // map[string]int 相当
anyMap := g.MapAny()                       // ゆるいパススルー
ad := g.MapOf[int](g.NumberJSON())         // AnyAdapter 化（オブジェクトフィールド用）
```

---

### Presence と WithMeta
`ParseFromWithMeta` は値と Presence を返します。Presence は JSON Pointer -> ビット集合です。

```go
dm, err := goskema.ParseFromWithMeta(ctx, schema, goskema.JSONBytes(data))
if dm.Presence["/nickname"] & goskema.PresenceSeen == 0 { /* 欠落 */ }
if dm.Presence["/nickname"] & goskema.PresenceWasNull != 0 { /* null */ }
if dm.Presence["/nickname"] & goskema.PresenceDefaultApplied != 0 { /* default 補完 */ }

// オブジェクトの出力を presence に従って整形
out := goskema.EncodePreservingObject(dm)
```

詳細: `docs/tutorial.md`（Presence / Canonical / Preserving の節）を参照。

---

### ストリーミング
- `StreamParse(ctx, schema, io.Reader, ...)` で巨大入力を段階検証。
- DSL の Object/Array/Primitive はストリーミングの最適化が効きます。未対応スキーマは any 構築経路にフォールバックします（検証結果は同一、メモリ特性のみ差）。

---

### JSON Schema 生成
任意の Schema は `JSONSchema()` を実装します。オブジェクトの Required はキー昇順、Unknown ポリシーは `additionalProperties` に反映されます。

---

### レシピ集
- 単項必須: `Field("email", g.StringOf[string]()).Required()`
- 複数必須: `Require("id","email")`
- 未知キーを厳密に: `UnknownStrict()`
- 未知キーを捨てる: `UnknownStrip()`
- 未知キーを集約: `UnknownPassthrough("extra")`（`extra` は `MapAny()` などで受ける）
- デフォルト適用と欠落/null 判定: `ParseFromWithMeta` と Presence ビット
- オブジェクトを presence に従い再エンコード: `EncodePreservingObject`

---

### 参考
- ユーザーガイド: `docs/user-guide.md`
- リファレンス: `docs/reference.md`
- 拡張（Codec/Refine/Format）: `docs/extensibility.md`

