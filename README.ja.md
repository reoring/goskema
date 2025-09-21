# Goskema

高速・型安全なスキーマ/コーデック基盤（Go）。**Zod の実運用上の痛点**（unknown/duplicate/presence/巨大入力/機械可読エラー/契約配布など）を **Go 向けに実務最適化**して解決します。

* **Schema ↔ Codec** を中核に、**wire（JSON 等）↔ domain（Go 型）** を一貫した意味論で統合
* **Presence（欠落 / null / default 適用）** を全パスで機械的に追跡し、**Preserve エンコード**でそのまま再現
* **重複キー検出**、**unknown キーのポリシー（Strict/Strip/Passthrough）**、**DoS 対策（MaxBytes/Depth）**
* **ストリーミング検証**で巨大配列/深いネストに強い
* **JSON Pointer＋コード**の**機械可読エラー**を標準化（UI/監査/可観測性に直結）
* **JSON Schema エクスポート**と**OpenAPI（Kubernetes CRD）取り込み**で契約配布と互換チェック

### TL;DR

- 外部 JSON・巨大入力・厳密な契約運用なら goskema。
- 単一スキーマで検証／型投影／エラー／契約配布を統合。
- Presence・duplicate/unknown・ストリーミング・Issues(JSON Pointer) を標準化。

### 30秒ピッチ

標準でも「動く」は作れる。ただし要件が増え、寿命が伸びるほどロジックは分散・重複する。goskema は「スキーマ＝唯一の仕様」を中心に、検証・エラー・コーデック・ドキュメント・テストを一体化し、変更に強い土台を提供します。

---

## 目次

* [これが不要なとき / 必要なとき](#これが不要なとき--必要なとき)
* [最短デモ](#最短デモ-presence欠落nulldefaultと-preserve-出力)
* [特徴](#特徴)
* [Why goskema?（なぜ今これなのか）](#why-goskemaなぜ今これなのか)
* [encoding/json / validator と使い分け](#encodingjson--validator-と使い分け要点)
* [JSON Schema エクスポート / OpenAPI 取り込み](#json-schema-エクスポート--openapi-取り込み)
* [Webhook 実装例（HTTP / Kubernetes）](#webhook-実装例http--kubernetes)
* [クイックスタート](#クイックスタート)
* [ストリーミング（巨大配列を安全に）](#ストリーミング巨大配列を安全に)
* [Enforcement（重複キー・深さ・サイズ制限）](#enforcement重複キーデプスサイズ制限)
* [WithMeta / Presence（欠落・null・default の見分け方）](#withmeta--presence欠落nulldefault-の見分け方)
* [NumberMode Tips（精度と速度）](#numbermode-tips精度と速度の選択)
* [JSON ドライバの切替](#json-ドライバの切替encodingjson--go-json--jsonv2)
* [DSL の概要](#dsl-の概要)
* [価値訴求 / ROI の考え方](#価値訴求--roi-の考え方)
* [使わない判断基準（キル基準）](#使わない判断基準キル基準)
* [段階導入のすすめ（薄いロールアウト）](#段階導入のすすめ薄いロールアウト)
* [よくある誤解への回答（FAQ）](#よくある誤解への回答faq)
* [ステータス / ロードマップ](#ステータス--ロードマップ)
* [ベンチマーク（抜粋）](#ベンチマーク抜粋)
* [安定性ポリシー / 対応 Go バージョン](#安定性ポリシー--対応-go-バージョン)
* [ライセンス / 貢献](#ライセンス--貢献)
* [拡張ポイント（Codec / Refine / Format）](#拡張ポイントcodec--refine--format)
* [エラーモデル](#エラーモデル)

---

## これが不要なとき / 必要なとき

* **不要（encoding/json や validator で十分）**

  * 入力は小さく**信頼できる**（社内生成のみ等）、未知キーや重複キーを厳密に扱わない
  * 欠落 / null / default の区別や **PATCH 用の presence 追跡が不要**
  * 巨大配列 / 深いネストの**ストリーミング検証が不要**
  * エラーは**文字列で十分**（UI/監査へ機械可読に渡す必要がない）

* **必要（goskema の価値が最大化される）**

  * \*\*Presence（欠落 / null / default 適用）\*\*を厳密に扱い、**Preserve 出力**をしたい
  * **JSON レベルの重複キー検出**・**未知キーの明示ポリシー（Strict/Strip/Passthrough）**
  * 外部入力や巨大入力を\*\*段階検証（ストリーミング）\*\*して **DoS 耐性**を高めたい
  * **双方向変換（Codec）**や **JSON Schema 生成 / OpenAPI 取り込み**で**契約配布**したい
  * **JSON Pointer＋コード**付きの**機械可読エラー**を UI/監査に直結したい

---

## 最短デモ: Presence（欠落/null/default）と Preserve 出力

```go
// スキーマ（map向け）: nickname は default を持つ
obj, _ := g.Object().
  Field("name",     g.StringOf[string]()).Required().
  Field("nickname", g.StringOf[string]()).Default("anon").
  UnknownStrict().
  Build()

// 入力: nickname は null（欠落ではない）
dm, _ := goskema.ParseFromWithMeta(ctx, obj, goskema.JSONBytes([]byte(`{"name":"alice","nickname":null}`)))

// Presence で欠落/null/default適用の有無を判定可能
_ = (dm.Presence["/nickname"] & goskema.PresenceWasNull) != 0 // => true

// Preserve 出力: 欠落は欠落のまま、null は null のまま、defaultで埋まっただけの値は落とす
out := goskema.EncodePreservingObject(dm)
_ = out // => map[string]any{"name":"alice","nickname":nil}
```

### 最短デモ: JSON の重複キー検出（Path/Code 付き）

```go
obj := g.Object().MustBuild()
opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}
_, err := goskema.ParseFrom(ctx, obj, goskema.JSONBytes([]byte(`{"x":1,"x":2}`)), opt)
if iss, ok := goskema.AsIssues(err); ok {
  // 例: code="duplicate_key", path="/x"
  fmt.Printf("%s at %s\n", iss[0].Code, iss[0].Path)
}
```

---

## 特徴

* **Schema/Codec の統合（wire ↔ domain）**
* **明示的な意味論**（unknown / duplicate / presence）
* **ストリーミング/巨大入力に強い**
* **型安全＆拡張性**（format/codec 追加）

---

## Why goskema?（なぜ今これなのか）

\*\*標準だけでは埋めきれない“実務の痛点”\*\*に踏み込みます：

1. **重複キー検出**
   `encoding/json` は「最後の値が勝つ」で検出不可。goskema は**ストリーミング段階で検出**し、`path="/x" code="duplicate_key"` を返却。**監査・脆弱性対策・UI 連携**に有効。

2. **unknown キーポリシーの宣言と一貫性**
   `DisallowUnknownFields` は粒度が粗く、**Strict/Strip/Passthrough** の運用方針を**宣言で一括**適用できない。goskema は**宣言で統一**し、エラーは **JSON Pointer** 付き。

3. **Presence を機械的に追跡**
   欠落 vs null vs default の区別を Go だけで安全に運ぶのは煩雑。goskema は**全パスに presence ビット**を持ち、**PATCH/差分/Preserve 出力**に直結。

4. **ストリーミング検証で DoS 耐性**
   ただの `Decoder` 合成では**検証・型投影・エラー収集**の流れを安全に作るのが難しい。goskema は**巨大配列/深いネスト**を段階検証し、**MaxBytes/Depth/FailFast** を宣言で固定。

5. **機械可読なエラー**
   **予約コード＋JSON Pointer＋Hint** を備え、**UI ハイライト/監査/可観測性**を標準化。SLO 分析にも活きる。

6. **Codec（wire↔domain）＋ Schema 連携**
   Marshaler/Unmarshaler だけでは**型変換・検証・契約配布**が分断しがち。goskema は **Codec/Refine** と **JSON Schema 出力 / OpenAPI 取り込み**で**契約と実装**をつなぐ。

> 小さくて信頼できる入力は標準で十分。**外部由来・巨大・壊せない運用**では goskema の投資が回収できます。

---

## encoding/json / validator と使い分け（要点）

| 観点                        | goskema                         | encoding/json + validator      |
| :------------------------ | :------------------------------ | :----------------------------- |
| unknown（明示）               | UnknownStrict/Strip/Passthrough | `DisallowUnknownFields` 等で個別対応 |
| duplicate key             | **検出（ストリーミング）**                 | **非対応**（最後の値が勝つ）               |
| presence（欠落/null/default） | **追跡＋Preserve出力**               | **非対応**（近似は自前実装）               |
| ストリーミング                   | **対応**（段階検証・Enforcement）        | **非対応**（全読み込み前提）               |
| エラー表現                     | **JSON Pointer＋コード（Issues）**    | `ValidationErrors`/文字列中心       |

詳細比較は `docs/compare.md` を参照。

---

## JSON Schema エクスポート / OpenAPI 取り込み

```go
// JSON Schema 出力
sch, _ := userSchema.JSONSchema()
b, _ := json.MarshalIndent(sch, "", "  ")
fmt.Println(string(b))

// OpenAPI (Kubernetes CRD) 取り込み
s, _, _ := kubeopenapi.ImportYAMLForCRDKind(crdYAML, "Widget", kubeopenapi.Options{})
_ = s
```

**使いどころ**

* クライアント/LSP/CLI 向けの事前検証
* 型生成・フォーム生成・API ドキュメントの土台
* サービス間の契約配布と互換チェック

**注意点**

* Presence（欠落/null/default）や**重複キー検出は JSON Schema に落ちきりません**（ランタイムで担保）。
* Unknown ポリシーは次に対応：`UnknownStrict` → `additionalProperties:false`、`UnknownStrip/Passthrough` → `true`。
* `default` は注釈。**値の適用はクライアント次第**。
* Codec/Refine の一部はスキーマに反映できない場合あり（必要に応じ `x-goskema-*` 拡張で注記）。

**最小保存例**

```go
sch, _ := userSchema.JSONSchema()
b, _ := json.MarshalIndent(sch, "", "  ")
_ = os.WriteFile("schema.json", b, 0o644)
```

---

## Webhook 実装例（HTTP / Kubernetes）

**ポイント**

* **ストリーミング前提**：`http.Request.Body` を `goskema.JSONReader(r.Body)` へ直接
* **Issues を UI/監査へ**：`JSON Pointer＋コード`
* **unknown/duplicate の明快制御**
* **Presence が簡単**：`PresenceOpt{Collect:true}`

**HTTP 版**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    s := buildSchema() // g.Object()...UnknownStrict()...MustBuild()
    opt := goskema.ParseOpt{
        Presence:   goskema.PresenceOpt{Collect: true},
        Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
    }
    dm, err := goskema.ParseFromWithMeta(r.Context(), s, goskema.JSONReader(r.Body), opt)
    if err != nil {
        if iss, ok := goskema.AsIssues(err); ok {
            _ = json.NewEncoder(w).Encode(map[string]any{"issues": iss})
            return
        }
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    _ = json.NewEncoder(w).Encode(map[string]any{
        "ok": true, "canonical": dm.Value, "presence": dm.Presence,
    })
}
```

**Kubernetes ValidatingWebhook（AdmissionReview v1）**

```go
crd := mustRead("crd.yaml")
s, _, _ := kubeopenapi.ImportYAMLForCRDKind(crd, "Widget", kubeopenapi.Options{})
opt := goskema.ParseOpt{
    Presence:   goskema.PresenceOpt{Collect: true},
    Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
}
_, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(ar.Request.Object), opt)
resp := &admissionResponse{UID: ar.Request.UID, Allowed: err == nil}
if err != nil {
    if iss, ok := goskema.AsIssues(err); ok {
        resp.Allowed = false
        resp.Status = &status{Code: 400, Reason: "Invalid", Message: iss[0].Code + " at " + iss[0].Path}
    }
}
```

Kubernetes 連携の設計（unknown/list-type/int-or-string など詳細）は `docs/k8s.md` を参照。

---

## インストール

```bash
go get github.com/reoring/goskema
```

```bash
# v0 系を使う場合（推奨）
go get github.com/reoring/goskema@v0
```

---

## クイックスタート

```go
package main

import (
    "context"
    g "github.com/reoring/goskema/dsl"
    "github.com/reoring/goskema"
)

type User struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}

func main() {
    ctx := context.Background()
    userSchema := g.ObjectOf[User]().
        Field("id",    g.StringOf[string]()).Required().
        Field("email", g.StringOf[string]()).Required().
        UnknownStrict().
        MustBind()

    user, err := goskema.ParseFrom(ctx, userSchema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com"}`)))
}
```

より多くの例は `docs/user-guide.md` を参照。

- サンプル: [sample-projects/user-api](sample-projects/user-api/)
- 代表テスト: [dsl/zod_basics_test.go](dsl/zod_basics_test.go), [dsl/codec_zod_usecases_test.go](dsl/codec_zod_usecases_test.go)

---

## ストリーミング（巨大配列を安全に）

巨大配列を“丸ごと展開せずに”1件ずつ検証・投影。`ParseFrom`/`StreamParse` が **Source 駆動で段階検証**します。

```go
// 要素スキーマ（型付き）
type Item struct{ ID string `json:"id"` }
itemS := g.ObjectOf[Item]().
  Field("id", g.StringOf[string]()).Required().
  UnknownStrip().
  MustBind()

// 入力を io.Reader として受け取り、Source 化
items, err := goskema.StreamParse(context.Background(), g.Array[Item](itemS), r)
_ = items; _ = err
```

* 失敗要素の `Issue.Path` は `/0`, `/2` のように**インデックス単位**で付与

* **現行仕様**：要素に 1 件でも失敗があると値は返さず `Issues` のみ返却
  → 成功要素を保持しつつエラー収集する逐次 API（Iterate/Handler 形式）は**将来提供予定**

* 実例: `benchmarks/benchmark_parsefrom_test.go` の `Benchmark_StreamParse_HugeArray_Objects`

* テスト: `dsl/array_stream_integration_test.go`

**未対応スキーマの扱い**

* DSL の Object/Array/Primitive は最適化対象。Union/カスタム Codec/`MapAny` 等は**従来経路へ自動フォールバック**（検証結果は同じ・大規模入力ではメモリ増）
* 必要に応じ **MaxBytes / FailFast / MaxDepth** で安全弁を設定
* ロードマップは `docs/eliminate-any.md` 参照

---

## Enforcement（重複キー・デプス・サイズ制限）

`ParseFrom`/`StreamParse` は内部で **重複キー検出・ネスト深さ・最大バイト数**をストリーミング検査。
`Issues` は **JSON Pointer** で正確なロケーション（例：`/items/0/foo`）。

最小コード例:

```go
opt := goskema.ParseOpt{
    Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
    MaxDepth:   16,
    MaxBytes:   1 << 20,
    FailFast:   true,
}
_, err := goskema.StreamParse(ctx, someArraySchema, r, opt)
_ = err
```

---

## WithMeta / Presence（欠落・null・default の見分け方）

**Presence は、各パスごとに「入力に現れたか / null だったか / default が適用されたか」を記録するフラグ集**です。

```go
dm, err := goskema.ParseFromWithMeta(ctx, user, goskema.JSONBytes(data))
if dm.Presence["/nickname"]&goskema.PresenceSeen == 0 {
    // 欠落（missing）
}
if dm.Presence["/nickname"]&goskema.PresenceWasNull != 0 {
    // null 入力
}
if dm.Presence["/nickname"]&goskema.PresenceDefaultApplied != 0 {
    // default で補われた
}
```

* 収集範囲は `PresenceOpt`（`Include`/`Exclude`/`Collect`）
* パス最適化は `PathRenderOpt`（`Intern`/`Lazy`）
* `EncodePreservingObject/Array` が **欠落/null/default** を再現

  * **注意**：`EncodePreserve` は presence が**必須**。`EncodeWithMode(..., EncodePreserve)` 単体呼び出しはエラー（`ErrEncodePreserveRequiresPresence`）

---

## NumberMode Tips（精度と速度の選択）

```go
// 既定: NumberJSONNumber（精度重視）
v1, _ := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))

// 速度優先: float64 丸め
src := goskema.WithNumberMode(goskema.JSONBytes(js), goskema.NumberFloat64)
v2, _ := goskema.ParseFrom(ctx, s, src)
```

* 精度維持（巨大整数/金額）: `NumberJSONNumber`（既定）
* 速度優先/軽量: `NumberFloat64`
  詳細は `dsl/numbermode_integration_test.go` と `docs/user-guide.md` を参照。

---

## JSON ドライバの切替（encoding/json ↔ go-json ↔ json/v2）

```go
// go-json を明示使用
import (
    goskema "github.com/reoring/goskema"
    drv "github.com/reoring/goskema/source/gojson"
)

func init() {
    goskema.SetJSONDriver(drv.Driver())
}
```

* ビルドタグ `-tags gojson`：`goccy/go-json` バックエンド
* 既定：`encoding/json` フォールバック
* **副作用 import** で既定ドライバ化：`import _ "github.com/reoring/goskema/source"`

`encoding/json/v2` を使う場合（実験的）：

```go
// 要: GOEXPERIMENT=jsonv2 と -tags jsonv2
import (
    goskema "github.com/reoring/goskema"
    drv "github.com/reoring/goskema/source/jsonv2"
)

func init() {
    goskema.SetJSONDriver(drv.Driver())
}
```

既定へ戻すには `goskema.UseDefaultJSONDriver()`。

**JSON Schema との整合性（UnknownStrip）**

* `UnknownStrip` はランタイムで「未知キーを受理してから投影で破棄」
* 生成 Schema は `additionalProperties:true`（受理）

  * IDE など**事前バリデーション**と**ランタイム挙動**の齟齬を避ける方針
  * **エンドツーエンドで拒否**したい場合は `UnknownStrict`（Schema も `false`）

---

## DSL の概要

```go
user := g.ObjectOf[User]().
  Field("id",    g.StringOf[string]()).Required().
  Field("email", g.StringOf[string]()).Required().
  UnknownStrict().
  MustBind()
```

詳細は `docs/dsl.md` へ。

---

## 価値訴求 / ROI の考え方

* **障害削減**：重複キー・unknown による**設定誤解/脆弱化**を**早期拒否**＋**正確なパス提示**で抑止
* **開発効率**：Presence を自前管理する**ボイラープレートの削減**（PATCH 合成・差分処理の安定化）
* **契約運用**：**JSON Schema 出力 ↔ OpenAPI 取り込み**で**契約配布と互換チェック**を自動化
* **SRE/セキュリティ**：**MaxBytes/MaxDepth** 等の **Enforcement を宣言で固定**し、DoS 経路を閉鎖

**簡易試算**（例）

* 外部 JSON 受信での不正入力率 × 解析・再配布・ロールバック時間（人件費）
* 運用年数 × 変更頻度 × フォーマット逸脱の検知漏れコスト
* ボイラープレート削減 LOC × バグ混入率 × 保守年数
  → **境界 10 エンドポイント**を goskema 化し、**月 1 件のインシデント回避**でも十分にペイするケースが多い。

---

## 使わない判断基準（キル基準）

* 入力が **社内生成のみ**で信頼でき、サイズも小さい
* PATCH や差分管理を **行わない**
* エラーは **ログ文字列だけ**で十分（UI/監査連携不要）

この条件では **`encoding/json`＋軽い validator** で十分で、goskema はオーバーエンジニアリングです。

---

## 段階導入のすすめ（薄いロールアウト）

1. **1 箇所だけ**：外部入力の **Validating Webhook**（Kubernetes なら Admission）を goskema 化
2. **重複/unknown/presence の検出実績**をログで可視化（`path`, `code`）
3. **Preserve 出力**で PATCH フローの不具合を **1 件でも**潰して事例化
4. **JSON Schema を配布**し、IDE/LSP で**事前検証**を回す
5. 効果が見えたら巨大配列/設定 JSON の経路へ拡張

**社内向け短文テンプレ**

> 標準だけで行けるのは「小さく信頼できる入力」。当社の外部 JSON は、
>
> * **duplicate/unknown をストリーミングで検出**
> * **欠落/null/default を機械的に区別（PATCH/Preserve）**
> * **JSON Pointer＋コードで UI/監査に直結**
>   が必要。goskema はここを標準化します。**簡単な所は現状維持**、**境界だけ差し替え**で薄く導入します。

---

## よくある誤解への回答（FAQ）

**Q.「標準だけで十分では？」**
A. 小さく信頼できる入力なら Yes。**重複キー・unknown・presence・巨大入力・機械可読エラー**が要ると No。goskema はここを**宣言で一括**かつ**実装と契約を統合**します。

**Q. validator ライブラリと何が違う？**
A. 多くは**構造/型検証と文字列エラー**中心。goskema は **JSON Pointer＋コード**で**UI/監査直結**、**Presence/Preserve**、**ストリーミング検証/Enforcement**、**Codec/Schema 連携**まで含めて**ランタイム運用**を作ります。

**Q. OpenAPI / JSON Schema で十分では？**
A. 契約配布には最適。ただし **Presence/duplicate/Strip/Preserve** などの**ランタイム意味論**は落ちます。goskema は\*\*契約（Schema）と実装（ランタイム）\*\*を橋渡しします。

**Q. CUE との違いは？**
A. CUE は強力な宣言言語。一方、**Go 型投影／Webhook 実装／Codec/Refine と統合したランタイム**という観点では goskema の方が**Go 実装寄りの落としどころ**です。併用も可能です。

**Q. パフォーマンスは？**
A. 素の Unmarshal より検証コストは当然増えますが、**ストリーミング最適化**により**巨大入力でも安定**。`docs/benchmarks.md` を参照。

---

## ステータス / ロードマップ

* 現在は **Interpreted エンジンの安定化**を優先
* **Compiled（コード生成）経路**は計画中（詳細は今後公開予定のユーザー向け資料に統合）

---

## ベンチマーク（抜粋）

|               Suite | Case                            | 備考                                                                               |
| ------------------: | :------------------------------ | :------------------------------------------------------------------------------- |
| compare / HugeArray | go-json/sonic/encoding/json と比較 | goskema の ParseAndCheck は厳密検証を含むため素の Unmarshal より遅くなるのが自然。ただし**巨大配列**でも**安定処理**。 |

実行方法（抜粋）:

```bash
make bench BENCH_FILTER=.
cd benchmarks/compare && go test -bench . -benchmem
```

---

## 安定性ポリシー / 対応 Go バージョン

* v0 系は仕様磨き期間として、必要な破壊的変更を認めます。変更時は **`CHANGELOG.md` と `docs/adr/`** に根拠と移行ガイドを記載。
* 最小サポート Go バージョンは `go.mod` に準拠（現時点: Go 1.25.1）。CI は `-race` を含むテストを継続実行。

---

## ライセンス / 貢献

* ライセンスはリポジトリの **LICENSE** を参照
* Issue/PR 歓迎。開発ポリシーは `docs/policy.md` を参照

---

## 拡張ポイント（Codec / Refine / Format）

```go
// RFC3339 string <-> time.Time のような型間変換（Codec）
c := codec.TimeRFC3339()
s := g.Codec[string, time.Time](c)
// Decode
t, _ := s.Parse(ctx, "2024-01-15T10:30:00Z")
// Encode
wire, _ := c.Encode(ctx, t)
```

詳細は `docs/extensibility.md`。

---

## エラーモデル

goskema は検証失敗を **Issues（\[]Issue）** としてまとめ、`error` を実装。各 `Issue` は：

* **Path**: JSON Pointer（例: `/items/2/price`）
* **Code**: 予約コード（例: `invalid_type`, `required`, `unknown_key`, `duplicate_key`, `too_small`, `too_big`, `too_short`, `too_long`, `pattern`, `invalid_enum`, `invalid_format`, `discriminator_missing`, `discriminator_unknown`, `union_ambiguous`, `parse_error`, `overflow`, `truncated`）
* **Message**: ローカライズ可能
* **Hint / Cause / Offset / InputFragment**: 任意の補助情報

```go
u, err := goskema.ParseFrom(ctx, userSchema, goskema.JSONBytes(input))
if iss, ok := goskema.AsIssues(err); ok {
    for _, it := range iss {
        log.Printf("%s at %s: %s", it.Code, it.Path, it.Message)
    }
    return
}
```

最小 JSON 例:

```json
{"issues":[{"code":"duplicate_key","path":"/x","message":"..."}]}
```

* **Fail-Fast** は `ParseOpt{FailFast:true}`、既定は **Collect**（複数収集）
* エラー順序は安定化（オブジェクトはキー名昇順、配列はインデックス昇順）
* 参考: `docs/error-model.md`、サンプル `examples/error-model/main.go`、テスト `api_error_model_test.go`

---

> 必要十分に軽い所は標準で、**壊したくない境界**は goskema で。
> **契約と実装**、**可観測性と運用**を繋いで、長期の現場を安定させます。

---

**Appendix: ミニ実例（誤解しがちな 2 点）**

**A. 重複キー**

* `encoding/json`：`{"x":1,"x":2}` → `{x:2}`（検出不可）
* goskema：`code="duplicate_key", path="/x"` を返却（監査/UI 直結）

**B. 欠落 vs null vs default**

* 標準：安全に区別するには自前工夫が必要
* goskema：`Presence` で `Seen/WasNull/DefaultApplied` を判定し、**Preserve 出力**で意図通りに再現

