## goskema リファレンス

主要なエラー語彙と、パース/DSL の各種オプションを一覧形式でまとめます。

### 用語と前提
- wire 型: JSON 入出力時のワイヤ型（string/bool/number/object/array など）。
- domain 型: アプリケーションの型（例: `type UserID string`）。
- `*Of[T]`: wire 型を関数名で明示し、domain 型 `T` に射影します（例: `StringOf[UserID]()`）。
- `~` 制約: `T ~string` は「基底が string の名前付き型（別名型）も許容」を意味します。

### DSL リファレンス（早見）
- プリミティブ
  - `String()`: `Schema[string]` を返す最小実装
  - `Bool()`: `Schema[bool]`
  - `NumberJSON()`: `Schema[json.Number]`（既定は文字列からの強制変換なし）
- 型投影（wire → domain）
  - `StringOf[T ~string]()` / `BoolOf[T ~bool]()` / `NumberOf[T ~string]()`
  - 数値ショートハンド（json.Number → 各ビット幅/型）
    - `IntOf[T ~int]()` / `Int32Of[T ~int32]()` / `Int16Of[T ~int16]()` / `Int8Of[T ~int8]()`
    - `UintOf[T ~uint64]()` / `Uint32Of[T ~uint32]()` / `Uint16Of[T ~uint16]()` / `Uint8Of[T ~uint8]()`
    - `FloatOf[T ~float64]()`
  - `SchemaOf[T](schema)`: 任意スキーマを T にラップ
- オブジェクト
  - `Object()`: ビルダー開始 → `Field`/`Require`/`Unknown*` → `Build/MustBuild`
  - `ObjectOf[T]()`: 型付きビルダー → 最後に `Bind/MustBind`
- 配列
  - `Array(elem)`: 要素スキーマから配列（`.Min/.Max` など制約可）
  - `ArrayOf[E](elem)`: フィールド用アダプタ（制約なし）
  - `ArrayOfSchema[E](builder)`: フィールド用アダプタ（制約あり）
- マップ
  - `Map(elem)`: 値スキーマ付き map
  - `MapOf[V](elem)`: フィールド用アダプタ
  - `MapAny()`: 疎な passthrough map

### エラー語彙（主なコード）
- invalid_type: 期待した型/構造ではない
- required: 必須プロパティが不足
- unknown_key: 未知のキーが含まれる（UnknownStrict時）
- duplicate_key: JSONオブジェクトのキー重複
- too_short / too_long: 長さ制約違反（配列・文字列）
- too_small / too_big: 範囲制約違反（数値など）
- pattern: 正規表現不一致
- invalid_enum: 列挙に含まれない
- invalid_format: 形式（format）不一致
- discriminator_missing / discriminator_unknown: 判別子不足/未知
- union_ambiguous: 非判別Unionで複数一致
- parse_error: パース時の一般エラー
- overflow: 桁あふれ・精度喪失
- truncated: 打ち切り（MaxIssues/MaxBytes など）

補足（Issues の構造）:
- `type Issues []Issue` は `error` を実装する集合型です（`errors.As(err, &issues)` で取り出し）。
- `Issue` は主に `Path`（JSON Pointer）, `Code`, `Message`, `Hint?`, `Cause?`, `Offset?`, `InputFragment?` を持ちます。
```go
if iss, ok := goskema.AsIssues(err); ok {
    for _, it := range iss {
        fmt.Printf("%s at %s: %s\n", it.Code, it.Path, it.Message)
    }
}
```

### Parse/Stream オプション
`ParseFrom`, `StreamParse` に渡す `goskema.ParseOpt` の主な項目:

- Strictness
  - OnDuplicateKey: オブジェクト内のキー重複をどう扱うか（例: `goskema.Error` でエラー化）
- MaxDepth: ネスト深さの上限
- MaxBytes: 入力の最大バイト数（ストリーミング時に有効）
- Presence: メタ情報収集の挙動
  - `PresenceOpt{Collect: false}` で収集無効化

利用例:
```go
opt := goskema.ParseOpt{
    Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
    MaxDepth:   8,
    MaxBytes:   1<<20,
}
_, err := goskema.StreamParse[struct{}](ctx, someSchema, r, opt)
```

### Number スキーマ（NumberJSON/NumberOf/Int*/Uint*/FloatOf）
`NumberJSON()` は既定で文字列からの強制変換を行いません。必要時のみ明示的に有効化します。
```go
n2 := g.NumberJSON().CoerceFromString()
v, err := n2.Parse(ctx, "1.0") // => json.Number("1")
```

`NumberOf[T]` は `json.Number` を経由して「数値を文字列表現として保持する」型 `T(~string)` に射影します。
`IntOf/Int32Of/Int16Of/Int8Of` は整数に、`Uint*/Uint32/Uint16/Uint8` は非負整数に、`FloatOf` は浮動小数に、それぞれ `json.Number` から投影します。
```go
type Price string
p := g.NumberOf[Price]()

// 整数に投影
age := g.IntOf[int]()
small := g.Int16Of[int16]()
code := g.Uint32Of[uint32]()
score := g.FloatOf[float64]()
```

### Unknown の扱い（オブジェクト）
- `UnknownStrict()`: 未知キーを禁止
- `UnknownPassthrough(fieldName)`: 指定フィールドに未知キーを透過（ビルド時に存在/型を検証）

### Presence 収集
`ParseFromWithMeta` / `DecodeWithMeta` は既定で Presence を収集します。
```go
dm, err := goskema.ParseFromWithMeta(ctx, schema, src)
_ = dm.Presence // nil ではない

dm2, err := goskema.ParseFromWithMeta(ctx, schema, src, goskema.ParseOpt{
    Presence: goskema.PresenceOpt{Collect: false},
})
```

### Bind のキー解決規則
「DSL明示 > `goskema:"name=..."` > `json`タグ名 > フィールド名」の優先で解決します。

### JSON Schema 出力
- 主要スキーマは `JSONSchema() (*js.Schema, error)` を実装します。
  - 例: `g.String().JSONSchema()`、`g.Object().Field("id", g.StringOf[string]()).MustBuild().JSONSchema()`。
- `SchemaOf[T](schema)` 経由の場合、JSON Schema は元の `schema` に基づきます。
- 注意点:
  - `MapAny()` は `additionalProperties: true`（厳密な型付けは行いません）。
  - カスタム Codec/複雑な制約は JSON Schema に完全には反映されない場合があります。

### 関連ドキュメント
- ユーザーガイド: `docs/user-guide.md`
- チュートリアル: `docs/tutorial.md`
- エラーモデル: `docs/error-model.md`

### DSL の型付けショートハンド `*Of[T]`
スキーマをドメイン型に射影するには `*Of[T]` ヘルパを利用します（ADR-0017）。`Adapt&#91;T](...)`/`AdaptWithDefault` は v0 開発期間中に削除済みです。

```go
// string wire -> domain T(~string)
g.StringOf[UserID]()

// bool wire -> domain T(~bool)
g.BoolOf[Flag]()

// number wire (json.Number) -> domain T
g.NumberOf[Price]()
g.IntOf[int]()
g.Uint32Of[uint32]()
g.FloatOf[float64]()

// object builder（ObjectTypedの別名）
g.ObjectOf[User]() // = g.ObjectTyped[User]()
```
備考:
- `StringOf[T]`/`BoolOf[T]` は `T` が `~string`/`~bool` の基底型制約を満たす必要があります。
- `NumberOf[T]` は `T` が `~string`（`json.Number` は内部的に string）である必要があります。
- 任意スキーマをラップする場合は `SchemaOf[T](schema)`、制約付き配列は `ArrayOfSchema[E](builder)` を利用できます。
- 旧 `Adapt&#91;T]` API は削除済みです。既存コードは `*Of`/`SchemaOf`/`ArrayOfSchema` への移行が必要です。

- `~` の意味: `T ~string` / `T ~bool` は「基底型が string/bool の任意の名前付き型（別名型）を許容する」ことを表します。
  - 例: `type UserID string` は `~string` を満たすので `g.StringOf[UserID]()` が使えます。
  - 生の型にも利用可: `g.StringOf[string]()` / `g.BoolOf[bool]()`。

```go
type UserID string

Field("id", g.StringOf[UserID]())  // OK: UserID は ~string
Field("id", g.StringOf[string]())  // 生の string にも対応
```

- 命名の意図: `StringOf[T]`/`BoolOf[T]`/`NumberOf[T]` は「wire 型（string/bool/number）を関数名で明示し、domain 型 `T` へ投影する」ことを示します。
  - `Of[T]` や `PrimitiveOf[T]` のように wire 型を隠すと曖昧になり、誤用の温床になります。
  - `g.String().As[T]()` のような形は、Go がメソッドに型パラメータを許可していないため採用できません（ADR-0017 参照）。



