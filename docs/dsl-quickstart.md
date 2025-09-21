## DSL Quickstart / チートシート（presence / unknown / duplicate）

最短5分で goskema の DSL を把握し、実運用でつまずきがちな unknown/duplicate/presence を一望できる早見表です。詳細は `docs/dsl.md` を参照してください。

### 0) 最小例（型付き + 厳密な unknown）

```go
type User struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}

user := g.ObjectOf[User]().
  Field("id",    g.StringOf[string]()).Required().
  Field("email", g.StringOf[string]()).Required().
  UnknownStrict().
  MustBind()

val, err := goskema.ParseFrom(ctx, user, goskema.JSONBytes([]byte(`{"id":"u_1","email":"a@x"}`)))
_ = val; _ = err
```

---

### 1) unknown の早見（Strict / Strip / Passthrough）

- **Strict（既定）**: 未知キーは `unknown_key` エラー
- **Strip**: 未知キーは受理しつつ破棄
- **Passthrough(target)**: 未知キーを `target` フィールドへ集約（`map[string]any` 等で受ける）

```go
// Strict: エラー
objStrict := g.Object().
  Field("name", g.StringOf[string]()).
  UnknownStrict().
  MustBuild()

// Strip: 破棄
objStrip := g.Object().
  Field("name", g.StringOf[string]()).
  UnknownStrip().
  MustBuild()

// Passthrough: extra へ集約
objPass := g.Object().
  Field("name",  g.StringOf[string]()).
  Field("extra", g.SchemaOf[map[string]any](g.MapAny())).
  UnknownPassthrough("extra").
  MustBuild()
```

補足:
- Passthrough の `target` は AnyAdapter で受けられる必要があります（例: `MapAny()`）。
- Union と併用時の規則は `docs/dsl.md` の Unknown/Passthrough 節を参照。

---

### 2) duplicate（重複キー）検出の早見

入力ストリーム段階で検出します。`ParseOpt.Strictness.OnDuplicateKey` に `Warn`/`Error` を指定。

```go
opt := goskema.ParseOpt{
  Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
}
_, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes(js), opt)
// 例: issue.code == "duplicate_key", path は JSON Pointer
```

注意:
- YAML → JSON 変換経由の入力では重複が失われる場合があります。JSON 直入力での検出が確実です。

---

### 3) presence の早見（欠落・null・default の判別）

`ParseFromWithMeta` は値と presence を返します。presence は `map[string]PresenceBit`（JSON Pointer をキー）です。

```go
dm, err := goskema.ParseFromWithMeta(ctx, schema, goskema.JSONBytes(js))
if dm.Presence["/nickname"] & goskema.PresenceSeen == 0 {
    // 欠落（入力に現れていない）
}
if dm.Presence["/nickname"] & goskema.PresenceWasNull != 0 {
    // null が入力された
}
if dm.Presence["/nickname"] & goskema.PresenceDefaultApplied != 0 {
    // default により補完された
}

// presence に従って“差分を保持した”出力を再構成
out := goskema.EncodePreservingObject(dm)
_ = out
```

Tips:
- 既定で presence を収集します。範囲は `ParseOpt.Presence`（Include/Exclude/Collect）で制御可能。
- 配列は `EncodePreservingArray(decoded)` を利用できます。

---

### 4) Required / Default / Optional の早見

```go
// 単項必須は Field(...).Required()
obj := g.Object().
  Field("email", g.StringOf[string]()).Required().
  MustBuild()

// 複数必須は Require(...)
obj2 := g.Object().
  Field("id",    g.StringOf[string]()).
  Field("email", g.StringOf[string]()).
  Require("id", "email").
  MustBuild()

// Default は欠落時に補完（presence に DefaultApplied が立つ）
obj3 := g.Object().
  Field("active", g.BoolOf[bool]()).Default(true).
  MustBuild()
```

---

### 4.5) 数値ショートハンド（Int*/Uint*/Float）

```go
user := g.ObjectOf[struct{ Age int `json:"age"`; Score float64 `json:"score"` }]().
  Field("age",   g.IntOf[int]()).Default(18).
  Field("score", g.FloatOf[float64]()).Default(0.0).
  UnknownStrict().
  MustBind()
```

---

### 5) NumberMode（精度 vs 速度）

```go
// 既定: JSONNumber（精度維持）
v, _ := goskema.ParseFrom(ctx, schema, goskema.JSONBytes(js))

// 速度優先: float64 丸め
src := goskema.WithNumberMode(goskema.JSONBytes(js), goskema.NumberFloat64)
v2, _ := goskema.ParseFrom(ctx, schema, src)
_ = v; _ = v2
```

---

### 6) ストリーミング（巨大配列を安全に）

```go
type Item struct{ ID string `json:"id"` }
elem := g.ObjectOf[Item]().
  Field("id", g.StringOf[string]()).Required().
  UnknownStrict().
  MustBind()

vals, err := goskema.StreamParse(ctx, g.Array[Item](elem), r)
_ = vals; _ = err
```

メモ:
- Object/Array/Primitive はストリーミングの最適化が効きます。未対応スキーマは any 経路へフォールバック（検証結果は同じ、メモリ特性のみ差）。

---

### 7) よく使うスニペット集（一行レシピ）

- **未知キーを厳密に**: `UnknownStrict()`
- **未知キーを捨てる**: `UnknownStrip()`
- **未知キーを集約**: `UnknownPassthrough("extra")`（`extra` は `MapAny()` で受ける）
- **単項必須**: `Field("email", g.StringOf[string]()).Required()`
- **複数必須**: `Require("id","email")`
- **default 補完**: `Field("active", g.BoolOf[bool]()).Default(true)`
- **presence 取得**: `ParseFromWithMeta(ctx, s, src)` → `dm.Presence`
- **重複キーをエラー**: `ParseOpt{Strictness: Strictness{OnDuplicateKey: Error}}`

---

### 関連ドキュメント
- 詳細リファレンス: `docs/dsl.md`
- ユーザーガイド: `docs/user-guide.md`
- エラーモデル: `docs/error-model.md`
- Union × Passthrough 規則: `docs/dsl.md`


