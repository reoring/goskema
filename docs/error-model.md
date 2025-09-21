## エラーモデル（ユーザー向けガイド）

goskema は検証失敗を Issues（[]Issue）として返します。各 Issue は JSON Pointer の Path と予約済み Code を持ち、UI/監査/ログに直結できる機械可読な形式です。

### 基本構造

```go
type Issue struct {
    Path           string // JSON Pointer（例: /items/2/price）
    Code           string // 例: invalid_type, required, unknown_key, duplicate_key, ...
    Message        string // ローカライズ可能
    Hint           string // 任意: 修正提案や補足
    Cause          error  // 任意: 根本原因
    Offset         int64  // 任意: 入力ソース上のバイト位置（不明時は -1）
    InputFragment  string // 任意: 入力断片
}

type Issues []Issue // error を実装
```

利用例:

```go
out, err := goskema.ParseFrom(ctx, schema, src)
if iss, ok := goskema.AsIssues(err); ok {
    for _, it := range iss {
        log.Printf("%s at %s: %s", it.Code, it.Path, it.Message)
    }
}
```

### 主なエラーコード（抜粋）

- invalid_type: 期待した型/構造ではない
- required: 必須プロパティが不足
- unknown_key: 未知のキー（UnknownStrict 時）
- duplicate_key: JSON の同一オブジェクト内でキーが重複
- too_short / too_long: 長さ制約違反（配列・文字列）
- too_small / too_big: 範囲制約違反（数値など）
- pattern: 正規表現不一致
- invalid_enum: 列挙に含まれない
- invalid_format: 形式検証に失敗
- discriminator_missing / discriminator_unknown: 判別キー不足/未知値
- union_ambiguous: 非判別 Union で候補が複数一致
- parse_error: パース時の一般エラー
- overflow: 桁あふれ・精度喪失
- truncated: 打ち切り（MaxIssues/MaxBytes 等）

### 表示と順序

- Path は JSON Pointer で安定化（オブジェクトはキー名昇順、配列はインデックス昇順）
- 大量エラー時は Issues に要約を持たせる設計（UI 側で先頭 N 件を表示など）

### 重複キー検出（duplicate_key）

JSON 入力ではストリーミング段階で重複キーを検出できます。`ParseOpt.Strictness.OnDuplicateKey` を `Error` にするとエラー化されます。

```go
opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}
_, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes(js), opt)
```

### Presence との関係（要点）

- WithMeta 系（`ParseFromWithMeta`/`DecodeWithMeta`）では `Decoded[T]` に Presence（欠落 / null / default 適用）を保持します
- Preserving 出力（`EncodePreserving`）は Presence に従って「出す / 出さない / null」を決定します

Presence と出力モードの詳細は `docs/tutorial.md` を参照してください。

### 参考

- ユーザーガイド: `docs/user-guide.md`
- DSL リファレンス: `docs/dsl.md`

