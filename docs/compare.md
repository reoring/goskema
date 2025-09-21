## ライブラリ比較（goskema / go-playground/validator / qri-io/jsonschema / CUE）

本ドキュメントは、実運用でよく問われる3観点「検証の厳密さ」「ストリーミング」「エラー表現」を中心に、goskema と主要な代替（go-playground/validator、qri-io/jsonschema、CUE）を比較します。加えて、実装スタイルや拡張性など周辺機能も整理しています。

### 比較サマリ

| 観点/ライブラリ | goskema | go-playground/validator | qri-io/jsonschema | CUE |
|:--|:--|:--|:--|:--|
| 検証の厳密さ（unknown/required ほか） | Object DSL で `UnknownStrict/UnknownStrip`、必須/任意/nullable/default を宣言。Presence と併用可。 | 構造体タグ中心。unknown は `encoding/json` 側の設定に依存。複雑な相互制約はカスタム実装で補う前提。 | JSON Schema に従う。`additionalProperties` や `required` など仕様ベースで厳密。 | 言語レベルで型・制約を統合。オブジェクトの「閉じる/開く」を表現でき、厳密なモデリングが可能。 |
| 重複キー検出 | ストリーミングで検出（`Strictness.OnDuplicateKey: Error`）。Path 付き Issue を返す。 | 非対応（Unmarshal 後の構造体検証のため JSON 重複は既に失われる）[1] | 典型構成では非対応（`encoding/json` 依存時は重複消失）[1] | 非対応（入力 JSON の重複は事前デコードに依存）[1] |
| Presence（欠落/null/default 適用の追跡） | `ParseFromWithMeta` が Presence を返す。`EncodePreserve` で欠落/null/default を保持した出力が可能。 | 明示サポートなし（ポインタ/ゼロ値/カスタム型で近似は可能）。 | 仕様上 `default` は注釈であり適用・追跡は非推奨/実装依存。標準 API では Presence 追跡なし。 | 評価結果から存在/非存在は表現可能だが、体系的な Presence キャリアは非標準。 |
| ストリーミング（io.Reader 直検証） | 対応。巨大入力でも段階検証。重複キー・深さ・サイズの Enforcement をエンジン内で実施。 | 非対応（メモリ内構造体の検証）。 | 一般に非対応（全体ロード後に検証）。 | 非対応（事前にデータを取り込み評価）。 |
| エラー表現 | JSON Pointer の Path、予約 Code、i18n 前提 Message、Hint/Offset/Fragment を保持。複数 Issue 収集と順序安定。 | `ValidationErrors`（`FieldError`）でフィールド単位。`Namespace/StructNamespace`、`Tag/Param` を保持。 | JSON Schema の慣例に近い `instanceLocation`/`keywordLocation` 等の位置情報を提供。 | CUE の診断情報（パスと理由）。JSON Pointer ではないがプログラマブルに取得可能。 |
| Go 型への投影 | DSL で型安全に投影（Codec/Refine 連携）。 | 構造体ベースで自然。 | 主に `any`/map に投影し検証。型連携は別途。 | CUE 定義 ↔ Go はツール・APIで橋渡し。 |
| JSON Schema 互換 | 生成/取込（Kubernetes CRD 由来の取り込みあり）。UnknownStrip のスキーマ注記方針を明記。 | 該当なし（スキーマ駆動ではない）。 | 準拠。ドラフト対応は実装バージョンに依存。 | スキーマ言語として別系統。必要なら JSON Schema へ変換レイヤが別途必要。 |
| 既定数値モード | `JSONNumber`/`float64` を切替可能（精度/速度）。 | 構造体側の型に従う。 | デコーダ依存。 | 評価系の数値表現に従う。 |
| 拡張性 | Codec（wire↔domain）、Refine、Format。 | カスタムバリデータ追加。 | 仕様拡張は JSON Schema 範囲。 | 言語機能で表現力が高い。 |

注記:

- [1] Go 標準の `encoding/json` は「同一キーの最後の値が勝つ」動作で、重複自体は維持されません。ライブラリが独自デコーダを持たない限り、重複キー検出は困難です。goskema は独自のストリーミング検査でこれを補っています。
- JSON Schema の `default` は注釈であり、バリデータが値を適用・保持することは標準では求められていません。

### もう少し詳しい比較

#### 検証の厳密さ（unknown/required/nullable/default）
- goskema: スキーマ宣言で意味論を明確化（`UnknownStrict/UnknownStrip`、必須/任意/nullable/default）。Presence と合成でき、PATCH/差分適用で威力を発揮。
- validator: フィールドタグ（例: `required`, `min`, `max`, `email`）。unknown キーは Unmarshal 側の `DisallowUnknownFields` 等で補助が必要。スキーマの再利用・配布は非主眼。
- qri-io/jsonschema: `additionalProperties`, `required`, `oneOf` 等を仕様どおりに解釈。厳密性はスキーマ次第で高い。
- CUE: オブジェクトの閉鎖/開放をモデリングでき、型・制約・定数・式を統一的に表現可能。

#### ストリーミング
- goskema: `ParseFrom`/`StreamParse` が `io.Reader` を直接処理。重複キー・深さ・最大バイト数などの Enforcement を検査しつつ段階的に検証・投影。
- 他 3 者: 一般に「全体を読み込んで」から検証する前提。巨大入力や DoS 耐性は周辺実装で担保する必要がある。

#### エラー表現
- goskema: JSON Pointer の Path、予約 Code（`invalid_type`, `required`, `unknown_key`, `duplicate_key`, ...）、メッセージ/i18n、Hint、入力フラグメント、オフセット等。複数エラーの収集・順序安定。
- validator: `ValidationErrors` と `FieldError`（`Namespace`, `Field`, `Tag`, `Param`, など）。翻訳パッケージあり。
- qri-io/jsonschema: `instanceLocation`/`keywordLocation`（JSON Pointer 互換の位置）等を返すスタイルが一般的。
- CUE: 診断エラーにパスと理由。エンコード次第で機械可読な構造化も可能だが標準形式は JSON Pointer ではない。

### 適材適所（選定の指針）

- goskema: Webhook、外部 API 入力、巨大 JSON の「厳密＋ストリーミング＋機械可読エラー」が要件のときに最適。Kubernetes 連携（CRD 取込 / int-or-string 方針）も提供。
- validator: アプリ内部の DTO/設定の「軽量な構造体バリデーション」。シンプルで導入容易。
- qri-io/jsonschema: 既存 JSON Schema の活用やスキーマ契約が中心のプロジェクト。IDE/LSP 連携や事前検証に親和的。
- CUE: 宣言的に構成や契約を統一したいとき。スキーマ記述力が高く、生成・検証・テンプレートを横断できる。

### 参考リンク

- go-playground/validator（エラーモデル・翻訳）: `https://github.com/go-playground/validator`
- qri-io/jsonschema: `https://github.com/qri-io/jsonschema`
- CUE（言語とツールリング）: `https://cuelang.org/`

### goskema との使い分け例

- 既存 JSON Schema 契約がある: まずは `kubeopenapi`/インポート機能で取り込み、ランタイムは goskema で厳密化（unknown/duplicate/PRESENCE）。
- アプリ内部の軽量チェック: `validator` を使うが、外部境界（Webhook/受信 API）は goskema で堅牢化。
- 設定ファイル群の全体モデリング: CUE で設計し、実行時の入力境界は goskema に寄せてエラー/i18n/Presence を活用。

---

補足: 本比較は代表的な実装スタイルを前提とした技術的観点の整理です。各ライブラリのリリースにより挙動や API が変化する可能性があります。詳細は各公式ドキュメントをご確認ください。


