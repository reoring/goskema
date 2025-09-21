## 拡張ガイド（format / codec / Refine）

### Format の追加（検証のみ）
```go
// init 時に登録する（最初の Parse 実行以降はレジストリがフリーズ）
func init() {
    goskema.RegisterFormat("hostname",
        func(b []byte) bool { /* fast-path（RE2想定）*/ return len(b) < 256 },
        func(s string) error { /* 厳密検証 */ return nil },
    )
}
```

### Codec の追加（A <-> B の双方向）
```go
// RFC3339 string <-> time.Time のような型間変換
type MyDomain struct{ time.Time }

func MyTimeCodec() goskema.Codec[string, MyDomain] {
    return codec.TimeRFC3339().Transform(
        func(t time.Time) (MyDomain, error) { return MyDomain{t}, nil },
        func(d MyDomain) (string, error) { return d.Time.Format(time.RFC3339Nano), nil },
    )
}
```

フィールドへの適用例:
```go
obj := g.Object().
  Field("start", g.SchemaOf[MyDomain](goskema.Codec[string, MyDomain](MyTimeCodec()))).
  Require("start").
  UnknownStrict().
  MustBuild()
```

### ユーザー定義検証（Refine）
```go
// 型付きスキーマに Transform/Refine 相当で後段検証を追加
s := g.String().Transform("nonempty", func(v string) (string, error) {
    if v == "" { return v, fmt.Errorf("empty") }
    return v, nil
})
```

注意:
- `Issue.Code` は `invalid_format`/`custom` 等を意図に応じて使用。
- 外部I/Oを伴う場合は `ctx` キャンセル対応を徹底。
- 登録系は並列安全ポリシー（ADR-0016）に従い、初回実行までに完了させる。


