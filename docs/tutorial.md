## goskema チュートリアル: Encode/Decode と Presence/Canonical 入門

このチュートリアルは、goskema の基本概念「Encode」「Decode」「Presence」「Canonical 出力（カノニカル）」を、初学者にもわかりやすく解説します。数字や時刻の正規化、optional/nullable/default の扱い、Preserving（プレザービング）出力との違いまで一気に把握できます。

### このチュートリアルのゴール
- **Encode/Decode** の役割が説明できる
- **Presence** が何を運ぶのか（欠落/null/default）の違いが理解できる
- **Canonical 出力** と **Preserving 出力** の違いと使い分けがわかる

---

## 最短レシピ（30秒で理解）

- **Decode**: 入力（wire 形式）をドメイン型に変換しつつ検証する。
- **Encode**: ドメイン型を出力（wire 形式）へ変換しつつ検証する。
- **WithMeta（DecodeWithMeta / ParseWithMeta）**: 値に加えて「どのフィールドが入力に現れたか（presence）」も一緒に運ぶ。
- **EncodePreserving**: WithMeta で得た presence に従い、入力の欠落/null をそのまま再現して出力する。
- **Encode（通常）」**: presence が無いので Preserving はできない。利用可能なのは **Canonical** のみで、Preserve を求める場合は WithMeta 経由（`EncodeWithDecoded` など）を使う。

例（概念）：
```json
// 入力（wire）: optional "age" は欠落、スキーマの default は 20 とする
{"name":"Alice"}

// WithMeta 経路（Preserving 出力）
// Decode 時に age=20 が補完されても、出力は「欠落を保持」する
{"name":"Alice"}

// 非 WithMeta 経路（Encode=Canonical）
// presence が無いので Canonical のみ。default を materialize した正規形を出力
{"age":20,"name":"Alice"} // キーは昇順で整列
```

---

## 用語と基本概念

### Decode（デコード）
- 方向: A(In/wire) -> B(Out/domain)
- 検証: `TypeCheck`（構造・型・presence/nullable/unknown-policy 等）→ `RuleCheck`（範囲/長さ/パターン/enum/Refine）
- 正規化: 必要に応じて `Coerce`/`Normalize` を行う。
- default: **Decode 時のみ適用**。optional 欠落に default があれば値を補う。
- 戻り値: 失敗すれば `error`。複数の検証失敗は `Issues`（`error` 実装）で返る。

### Encode（エンコード）
- 方向: B(Out/domain) -> A(In/wire)
- 検証: `Out.ValidateValue`（値が Out の規則を満たすか）→ 変換後に `In.Parse` で再検証。
- presence: **通常の `Encode` は presence を持っていない** ため、Preserving は不可能。Canonical のみサポートし、Preserve を要求すると `ErrEncodePreserveRequiresPresence` を返す。

### WithMeta（値＋presence を運ぶ経路）
- `ParseWithMeta` / `ParseFromWithMeta` / `DecodeWithMeta` は `Decoded[T]` を返す。
- `Decoded[T]` は `Value` と `PresenceMap`（JSON Pointer -> `Presence` ビット列）を持つ。
- `EncodePreserving(decoded)` は、`PresenceMap` に従って出力の「欠落/ null / default materialize」を決める。

### Presence（プレゼンス）とは
- 「その場所が入力に現れたか（seen）」「`null` だったか（wasNull）」「default が適用されたか（defaultApplied）」のビットを運ぶ軽量メタデータ。
- 公開 API 上のパスは常に「ドメイン形状」の JSON Pointer で表す（例: `/user/age`）。
- 代表的な違い：
  - 欠落 vs null の区別（欠落=seen=false, wasNull=false / null=seen=true, wasNull=true）
  - default の適用有無（defaultApplied=true で、Decode 時に値が補われたことがわかる）

---

## Canonical 出力（カノニカル）とは

presence 情報が無いときの既定の出力モード。機械的・安定的な「正規形」で表現します。主なルール：

- **オブジェクトのキー順**: UTF-8 バイト昇順でソート。配列は順序維持。
- **文字列**: 最小エスケープのみ（"、\\、制御文字）。HTML エスケープは無効（`<` はそのまま）。
- **数値**
  - 共通: `-0` は `0`。`NaN`/`±Inf` はエラー。先頭ゼロは禁止（0 以外）。
  - Float/JSONNumber: 指数は必要最小、小数の末尾 0 を除去（`1.2300`→`1.23`、`1.0`→`1`）。
  - Decimal: 指数禁止。末尾 0 を除去し、消えれば整数化（オプションで KeepScale）。
  - BigInt: 10 進プレーン表記のみ。
- **日時**: `time.Time` は UTC 正規化＋ RFC3339。小数秒は 1〜9 桁で末尾 0 を除去。
- **null/optional/default/空コレクション**
  - `null` はそのまま出力。
  - optional の欠落は、default があれば値を materialize して出力。無ければ欠落のまま。
  - 空配列/空オブジェクトは「存在している」なら `[]`/`{}` を出力。欠落は出力しない。

ポイント: 非 WithMeta の `Encode` は、この Canonical 出力のみを提供します（Preserve 指定は `ErrEncodePreserveRequiresPresence`）。

---

## Preserving 出力とは（WithMeta が必要）

`DecodeWithMeta` / `ParseWithMeta` で取得した `Decoded[T]` を `EncodePreserving` に渡すと、presence に従って「入力の痕跡（欠落/null/default）」を再現する出力になります。

- 欠落だった場所は欠落のまま。
- `null` だった場所は `null` のまま。
- `default` が適用されていても、欠落だったなら省略（スナップショット出力ではなく差分再現に適する）。

---

## Canonical と Preserving の違い

- **ひとことで**
  - **Canonical**: 「どう書くか（表記ルール）」を決める正規形。
  - **Preserving**: 「何を出すか（出す/出さない/null）」を presence で決める。

- **決定基準の違い**
  - **Canonical**: presence を使わない。スキーマ規則のみで判断。
    - optional 欠落 + default あり → 値を materialize
    - `null` はそのまま出力
    - キー順/数値/日時などは正規化（共通の表記ルール）
  - **Preserving**: presence を使う（`seen/wasNull/defaultApplied`）。
    - 欠落（`seen=false`）→ 出さない
    - `null`（`wasNull=true`）→ `null` を出す
    - default が適用されていても、入力が欠落なら出さない
    - 値の「書式」自体は Canonical と同じ（違うのは「出す/出さない」の判定）

- **心のモデル**
  - **Canonical** = 安定スナップショット（比較しやすい正規表記）
  - **Preserving** = 入力の痕跡を再現（差分や PATCH 向き）

- **ミニ例（同じ出力でも理由が違う／結果が分かれるケース）**
  1) optional + default=20、入力で `age` 欠落
     - Canonical: `{"age":20}`（正規形として default を materialize）
     - Preserving: `{}`（欠落を保持）
  2) 入力で `{"age":null}`
     - Canonical: `{"age":null}`（規則上 null はそのまま）
     - Preserving: `{"age":null}`（presence が `wasNull=true` なので再現）
  3) unknown passthrough + WithMeta
     - Preserving: unknown の presence も `seen=true` として再出力
     - Canonical: presence 非依存。ドメイン上の値があれば規則通りに出力

- **擬似ルール（イメージ）**
  - Preserving:
    - if `presence.seen == false` → omit
    - else if `presence.wasNull == true` → output `null`
    - else → output value（表記は Canonical と同じ）
  - Canonical:
    - if optional 欠落 and default あり → output default
    - else if 欠落 → omit
    - else → output value（表記は Canonical の規則）

- **使い分けのクイックガイド**
  - 部分更新・差分適用・PATCH API → Preserving
  - スナップショット・エクスポート・監査ログ → Canonical

---

## 実践シナリオで理解する

### 1) optional + default の典型例
前提: `age` は optional、default=20。

入力:
```json
{"name":"Alice"}
```

- WithMeta 経路（Preserving）
```json
{"name":"Alice"}
```

- 非 WithMeta（Encode=Canonical）
```json
{"age":20,"name":"Alice"}
```

解説: Decode では `age=20` が値として補われるが、Preserving は「欠落を保持」、Canonical は「正規形として materialize」。

### 2) null と欠落は別物（見た目が同じでも意味が違う）
入力:
```json
{"age":null}
```

- Preserving → `{"age":null}`（入力で null だった事実をそのまま再現）
- Canonical → `{"age":null}`（規則上、null はそのまま出力）

ここでのポイントは「表現は同じでも、意味付けの根拠が違う」ことです。
- Preserving の根拠: presence が `seen=true, wasNull=true` だから null を出す
- Canonical の根拠: 正規形の規則で「null はそのまま」と定まっているから null を出す

差が出るのは「欠落だった場合」や「default を materialize するか」の判断です（例「1) optional + default」を参照）。
また、どちらのモードでも書式ルール（キー順、数値・日時の正規化など）は同じです。Preserving は「出す/出さない」を presence で決めるだけで、出力の書式自体は Canonical と共通です。

### 3) 数値と時刻の正規化（Canonical）
入力:
```json
{"n":-0,"f":1.2300,"t":"2025-01-01T00:00:00+09:00"}
```

Canonical 出力:
```json
{"f":1.23,"n":0,"t":"2024-12-31T15:00:00Z"}
```

### 4) unknown キーの保持（Passthrough + Preserving）
`UnknownPassthrough` が有効で、余剰キーを `unknown_target`（例: `/meta/extra`）に集約する設定の場合、WithMeta 経路ではそれらの presence も `seen=true` として記録され、`EncodePreserving` で再出力されます。

---

## いつどちらを使う？（ガイド）

- **部分更新/差分適用/PATCH API**: WithMeta + `EncodePreserving` を使う（欠落/null の区別が重要）。
- **スナップショット/エクスポート/監査ログ**: 非 WithMeta の `Encode`（Canonical）で安定的な正規形を出力。
- **unknown キーを維持したい**: `UnknownPassthrough` と WithMeta を併用し、Preserving で再出力。

---

## 失敗時の扱い（error と Issues）

- すべての外部 API は `error` を返す設計。複数の検証失敗は `Issues` が `error` を実装して表現します。
- 呼び出し側は `errors.As(err, &goskema.Issues)` で詳細を取り出せます。

---

## 参考: 最小コード例（概念）

```go
// Codec 経路の WithMeta と Preserving
dx, err := ISO.DecodeWithMeta(ctx, "2025-01-01T00:00:00Z")
if err != nil { /* handle */ }
wire, err := ISO.EncodePreserving(ctx, dx)
```

```go
// 非 WithMeta（Canonical へフォールバック）
out, err := ISO.Encode(ctx, time.Now())
```

---

## さらに学ぶ
- エラーモデルの使い方: `docs/error-model.md`

このチュートリアルの内容は上記ドキュメントに準拠しています。実装の進展に伴い更新される可能性があります。
