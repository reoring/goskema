## Kubernetes CRD 互換設計（OpenAPI v3）

本書は、Kubernetes CRD の `openAPIV3Schema` を goskema で「実用的に」検証するための設計方針と仕様を示す。goskema の強み（unknown/duplicate/presence・ストリーミング・Codec統合）を活かしつつ、完全一致ではなく現実的互換を目指す。

### 目的
- CRD スキーマ（OpenAPI v3 + `x-kubernetes-*` 拡張）を goskema IR に取り込み、ローカル/CI/サイドカー/Webhook での高速かつ安定した検証を可能にする
- unknown の扱い、list 型の一意性／キー制約、presence、デフォルト適用の方針を明確化する
- 必要に応じて CEL（`x-kubernetes-validations`）をプラグインで評価できる拡張点を提供する

### 非目標
- Kubernetes apiserver の挙動（完全一致）を再現すること
- Admission webhook 連鎖や最終的なデフォルト適用順序の厳密再現
- すべての OpenAPI/JSON Schema 機能の損失なしラウンドトリップ

### 適用範囲
- OpenAPI 3.1（JSON Schema 2020-12 準拠）および 3.0（`nullable` 等を正規化して対応）
- Kubernetes 拡張: `x-kubernetes-preserve-unknown-fields`, `x-kubernetes-int-or-string`, `x-kubernetes-list-type`, `x-kubernetes-list-map-keys`, `x-kubernetes-embedded-resource`, `x-kubernetes-validations`（CEL）

## 基本方針（Compile / Delegate の二本立て）

- Compile（本命）
  - `openAPIV3Schema` を goskema の IR に「コンパイル」して評価
  - 長所: ストリーミング可、unknown/presence/duplicate を一級で制御、Codec と自然に統合
  - 注意: 合成（oneOf/anyOf/if-then-else 等）や `contains` はストリーミング時に遅延確定や一時バッファが必要

- Delegate（補完）
  - CEL など一部機能を外部実装に委譲（例: `cel-go`）。必要なら JSON Schema バリデータへの委譲モードも検討
  - 長所: 既存実装の互換性を活用
  - 注意: ストリーミングは限定的／非対応になる場面がある

## 互換マトリクス（要約）

| 項目 | 状態 | 備考 |
|---|---|---|
| 型（string/number/integer/boolean/null/object/array） | 対応 | 3.0 は `nullable` を Union に正規化 |
| 文字列（min/max/pattern/enum/const/format） | 対応（formatは注釈） | `format` の厳格度はオプション |
| 数値（minimum/maximum/exclusive*, multipleOf） | 対応 | NumberMode で丸め・表現を固定 |
| 配列（items, min/maxItems, uniqueItems, contains*） | 部分 | `contains*` はストリーミング時に遅延確定 |
| オブジェクト（properties, required, additionalProperties, patternProperties） | 対応 | `unevaluated*` は部分（オプションで近似） |
| 合成（allOf/anyOf/oneOf/not/if-then-else） | 部分 | 曖昧時バッファ戦略をオプション化 |
| `$ref`, `$defs` | 対応 | スキーマ内/外参照の解決器を提供 |
| `x-kubernetes-preserve-unknown-fields` | 対応 | Unknown: Prune/Strict/Preserve を切替 |
| `x-kubernetes-int-or-string` | 対応 | Union（integer ∪ string[pattern]）として扱う |
| `x-kubernetes-list-type`（atomic/set/map） | 対応 | set: 一意, map: `list-map-keys` で一意 |
| `x-kubernetes-list-map-keys` | 対応 | キー組で一意性検証 |
| `x-kubernetes-embedded-resource` | 対応（任意） | `apiVersion/kind/metadata` の存在・型検証 |
| `x-kubernetes-validations`（CEL） | 任意（プラグイン） | `cel-go` への委譲（オプション） |
| `default` | 任意 | Apply / Annotate / Ignore をモード化 |
| duplicate key 検出 | 対応（JSON入力時） | YAML→JSON 変換経路では要注意（後述） |

## マッピング仕様（要点）

- 型/基本制約
  - `type`, `enum`, `const`, `required`, `properties`, `items` 等は素直に IR へ写像
  - OpenAPI 3.0 の `nullable: true` は `type: [T, null]` の Union として扱う
  - 数値は `NumberMode` を尊重（浮動小数/整数の丸め・表現差異を吸収）

- unknown（未知フィールド）
  - `additionalProperties: false` → UnknownStrict
  - `x-kubernetes-preserve-unknown-fields: true` → UnknownPreserve（prune せず保持）
  - 既定は Kubernetes に合わせて Prune 相当（オプションで Strict/Preserve を選択）
  - 優先順位（反転不可・仕様固定）:
    1. `x-kubernetes-preserve-unknown-fields`
    2. `additionalProperties`
    3. `patternProperties`

- duplicate key（重複キー）
  - JSON ストリーム入力ではパース段階で検出可能（`error/last-wins/first-wins` を選択式）
  - YAML→JSON 変換を経た入力は重複が失われやすく、検出できない場合がある旨を文書化

- list 型
  - `x-kubernetes-list-type: set` → 要素一意（構造的同一性の定義は IR の等価比較に従う）
  - `x-kubernetes-list-type: map` + `x-kubernetes-list-map-keys: [k1, k2,...]` → 指定キー組で一意
  - `atomic` → 配列全体を原子的に扱う（差分適用やマージ方針は注釈）
  - 等価定義（set/map 共通の基準）:
    - `EqualMode` を導入して選択可能
      - `Structural`: 構造的等価（フィールド順は不問、内部配列順は値として比較）
      - `CanonicalJSON`: キーソート・正規化後の JSON 表現で比較
    - map 型ではキー抽出前に `required`/型検証を実施。欠落・型不一致は一意性検査をスキップし、先に該当の Issue を出す
  - 重複検出のヒント: 衝突時に `Hint` へ「最初の出現インデックス」を含める（例: `first at /spec/selectors/1`）

- `x-kubernetes-int-or-string`
  - Union 型として `integer ∪ string[pattern: ^-?\d+$]` を既定表現

- `x-kubernetes-embedded-resource`
  - `apiVersion`, `kind`, `metadata` の存在・型チェックを有効化するオプションを提供

- CEL（`x-kubernetes-validations`）
  - オプショナルプラグイン。`cel-go` へ式を渡し、`self` などのバインドを提供
  - 評価は Compile 結果に対する後段チェックとして実行し、エラーは `Issues` へ正規化
  - 実装ガイド: コンパイルキャッシュ、評価タイムアウト、関数ホワイトリスト、式サイズ上限、`self`/`apiVersion`/`kind`/`metadata` の束縛、`Code:"cel_violation"` と `rule.name`/短縮式を `Hint` に格納

- default 適用
  - `DefaultMode: Apply | Annotate | Ignore`
  - Apply: 欠落時に適用し投影値へ反映／Annotate: 値はそのまま、注釈として保持／Ignore: 参照のみ
  - 推奨パイプライン順序: Prune/Strict → Decode/Codec → Default Apply → Refine → CEL
  - Presence の記録: JSON Pointer 基準で `missing` / `wasNull` / `defaultApplied` を注釈として保持
  - Export 時の情報落ち軽減: `x-goskema-presence: true` を注釈出力（任意）

### Union の曖昧性（oneOf/anyOf）

- 既定: 曖昧（複数候補が一致）ならエラー（`union_ambiguous`）
- オプション: `Ambiguity: Error | FirstMatch | ScoreBest`（ScoreBest は制約違反数が最少のシグネチャを選択）
- CRD では運用上 `Error` 推奨

## ストリーミング戦略

- 逐次確定できるもの
  - 巨大配列（`items` の単一スキーマ）
  - オブジェクトの `required`／`additionalProperties`／`patternProperties`

- 遅延確定や一時バッファが必要なもの
  - `oneOf/anyOf/if-then-else` の曖昧解消
  - `contains/minContains/maxContains` の判定

- オプション
  - `StreamingStrategy` にバッファ閾値や曖昧時の確定ルールを設定（fail-fast / best-effort / full-buffer）
  - contains 系: ストリーミングでヒット数と該当インデックスを保持。`maxContains` 超過に対する「超過箇所の特定」は追加バッファが必要（閾値で制御）

## エラーモデル

- goskema の `Issues` を使用し、パス順・安定なメッセージ・収集/Fail-Fast の切替を提供
- Delegate（CEL など）で得たエラーも同一モデルへ正規化
- 表示互換: JSON Pointer に加え、K8s 風の `PathRenderOpt` により `.spec.replicas` 形式を選択可

## API スケッチ（案）

```go
package kubeopenapi

type UnknownBehavior int
const (
    UnknownPrune UnknownBehavior = iota
    UnknownStrict
    UnknownPreserve
)

type Profile string
const (
    ProfileStructuralV1 Profile = "structural-v1"
    ProfileLoose         Profile = "loose"
)

type DefaultMode int
const (
    DefaultIgnore DefaultMode = iota
    DefaultAnnotate
    DefaultApply
)

type Options struct {
    Profile           Profile         // 互換プロファイル。既定: structural-v1
    Unknown           UnknownBehavior // 既定: UnknownPrune（Kubernetes に準拠）
    EnableCEL         bool            // `x-kubernetes-validations` を有効化
    DefaultMode       DefaultMode
    StreamingStrategy goskema.StreamingStrategy
    NumberMode        goskema.NumberMode     // Kubernetes 用プリセットを提供（例: NumberFloat64）
    PathRender        goskema.PathRenderOpt  // K8s風パス併記などの表示制御
    EnableEmbeddedChecks bool                 // `x-kubernetes-embedded-resource` の最小検証を有効化
    EqualMode         goskema.EqualMode       // set/map の等価基準（Structural/CanonicalJSON）
    Ambiguity         goskema.Ambiguity       // Union の曖昧性戦略（Error/FirstMatch/ScoreBest）
}

type Diag interface {
    HasWarnings() bool
    Warnings() []string
}

// OpenAPI v3.1/3.0 のスキーマ（生 JSON / 構造体いずれも可）を IR へコンパイル
// 仕様ギャップは diag に警告として格納される（例: unevaluated* の近似など）
func Import(schema any, opts Options) (goskema.Schema, Diag, error)
```

使用例（概念）:

```go
data := loadCRDSchemaJSON()
sch, diag, err := kubeopenapi.Import(data, kubeopenapi.Options{
    Unknown:     kubeopenapi.UnknownPrune,
    Profile:     kubeopenapi.ProfileStructuralV1,
    EnableCEL:   true,
    DefaultMode: kubeopenapi.DefaultAnnotate,
    NumberMode:  goskema.NumberFloat64, // Kubernetes プリセット相当
    PathRender:  goskema.PathRenderK8s,
    Ambiguity:   goskema.AmbiguityError,
    EqualMode:   goskema.EqualStructural,
    EnableEmbeddedChecks: true,
})
if err != nil { /* handle */ }
if diag.HasWarnings() { /* optionally enforce zero-warn in CI */ }

out, issues := goskema.ParseFrom(reader, sch).Collect()
if issues.Any() { /* handle */ }
_ = out // 投影結果（Codec と組み合わせて wire→domain 変換も可能）
```

## 例（抜粋）

CRD スニペット（list-map-keys, int-or-string, preserve-unknown）:

```yaml
openAPIV3Schema:
  type: object
  x-kubernetes-preserve-unknown-fields: true
  properties:
    replicas:
      oneOf:
        - type: integer
        - type: string
          x-kubernetes-int-or-string: true
    selectors:
      type: array
      x-kubernetes-list-type: map
      x-kubernetes-list-map-keys: [name, namespace]
      items:
        type: object
        properties:
          name: { type: string }
          namespace: { type: string }
        required: [name, namespace]
```

主な対応：
- `x-kubernetes-preserve-unknown-fields: true` → UnknownPreserve
- `x-kubernetes-int-or-string` → `integer ∪ string[pattern: ^-?\d+$]`
- `x-kubernetes-list-type: map` + `x-kubernetes-list-map-keys` → 指定キー組で一意性検証

## テスト計画

- `testdata/crd/` に代表 CRD を配置し、以下を網羅
  - 正常ケース（最小/典型）
  - unknown の Prune/Strict/Preserve 各モード
  - list-type: set/map の一意性違反
  - `int-or-string` の両経路
  - 合成（oneOf/anyOf/if-then-else）の曖昧系
  - CEL（有効時/無効時）の通過・エラー
- 大規模配列・深いネストでのストリーミング動作とメモリフットプリントをベンチ比較（`benchmarks/`）
 - プロパティベーステストで一意性検証（順序・重複・欠落・型不一致のクロス）
 - contains 系の `min/maxContains` で境界条件（0,1,N-1,N,N+1）を網羅
 - Union 曖昧性の戦略差分（Error/FirstMatch/ScoreBest）での出力安定性

## 既知の差異・注意点

- duplicate key: JSON 直入力では検出可だが、YAML→JSON 変換後は失われる場合がある（CLI/SDK で警告）
- default 適用のタイミングは apiserver と一致しない。`DefaultMode` を明示設定すること
- `unevaluated*` は JSON Schema 2020-12 の意味論が広く、完全一致は困難。近似をオプションで提供
- `contains*` と合成の曖昧解消は、ストリーミング時に一時バッファが必要（閾値超過時のフォールバックを用意）

## YAML 入力互換（重複キー）

- YAML→JSON 変換経路では重複キーが失われることがある
- CLI/SDK には重複検出と行列情報付きのローダを提供する方針（例: `kubeopenapi.StrictYAMLReader`）

```go
r := kubeopenapi.StrictYAMLReader(bytes.NewReader(yamlBytes))
sch, diag, err := kubeopenapi.Import(r.AsJSONStream(), opts)
```

## 期待値と迂回策

- `unevaluated*`: 近似（UnknownStrict/Preserve と Refine の併用）で代替。完全一致は非目標
- `format`: 既定は注釈として扱い、厳格化はオプションで有効化
- duplicate key: JSON 入力では検出、YAML は Strict ローダの利用を推奨
- `default`: apiserver と適用タイミングが異なるため `DefaultMode` を明示。監査や差分用途には `Annotate` 推奨
- contains/合成の曖昧: ストリーミングでは閾値を越える場合に非ストリーミングにフォールバックする運用を推奨

## ロードマップ（段階導入）

1. Import（主要80%）: 型/enum/const/required/items/properties/additionalProperties/nullable/int-or-string
2. list-type（set/map）と list-map-keys、一意性検証の実装
3. `x-kubernetes-preserve-unknown-fields` と Unknown モード（Prune/Strict/Preserve）
4. 合成（oneOf/anyOf/if-then-else）の遅延確定とストリーミング戦略
5. CEL プラグイン（任意、`cel-go`）
6. 互換マトリクスの精緻化と E2E 例の拡充



