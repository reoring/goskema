# Kubernetes CRD Validator サンプル

goskema の kubeopenapi パッケージを使って、Kubernetes CRD の厳密な検証を行うサンプルです。

## 特徴

- **CRD スキーマ読み込み**: YAML から CRD 定義を読み込み、goskema スキーマに変換
- **厳密な検証**: AdmissionWebhook 風の厳密な検証
- **YAML/JSON 両対応**: Kubernetes リソースの一般的な形式をサポート
- **構造化エラー**: JSON Pointer での正確なエラー位置表示

## サンプル CRD

このサンプルでは、Widget という仮想的な CRD を使用します：

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              size:
                type: string
                enum: ["small", "medium", "large"]
              replicas:
                type: integer
                minimum: 1
                maximum: 10
              config:
                type: object
                additionalProperties: false
                properties:
                  timeout:
                    type: string
                  retries:
                    type: integer
            required: ["size", "replicas"]
        required: ["spec"]
  scope: Namespaced
  names:
    plural: widgets
    singular: widget
    kind: Widget
```

## 実行方法

```bash
# 依存関係のセットアップ
go mod init k8s-crd-validator-sample
go mod edit -replace github.com/reoring/goskema=../..
go mod tidy

# バリデーター実行
go run . validate widget.yaml

# または標準入力から
kubectl get widgets my-widget -o yaml | go run . validate -
```

## テスト例

### 正常な Widget リソース
```yaml
apiVersion: example.com/v1
kind: Widget
metadata:
  name: my-widget
  namespace: default
spec:
  size: medium
  replicas: 3
  config:
    timeout: "30s"
    retries: 3
```

### エラーが発生する例

#### 必須フィールドの欠如
```yaml
apiVersion: example.com/v1
kind: Widget
metadata:
  name: invalid-widget
spec:
  # size が欠如している
  replicas: 3
```

#### 値の範囲外
```yaml
apiVersion: example.com/v1
kind: Widget
metadata:
  name: invalid-widget
spec:
  size: medium
  replicas: 15  # 最大値10を超過
```

#### 未知フィールド（Kubernetes の structural schema に従い厳密に検証）
```yaml
apiVersion: example.com/v1
kind: Widget
metadata:
  name: invalid-widget
spec:
  size: medium
  replicas: 3
  unknown_field: "this will be rejected"
```

## 学習ポイント

1. **CRD 統合**: 実際の Kubernetes CRD との統合方法
2. **OpenAPI Schema**: OpenAPI v3 スキーマからの goskema スキーマ生成
3. **Kubernetes 特有制約**: structural schema、additionalProperties 等の扱い
4. **YAML 処理**: YAML 入力の正規化と検証
5. **CLI 作成**: コマンドライン版のバリデーター実装

## 応用例

このサンプルを参考に以下が作成可能：

- **AdmissionWebhook**: 実際の Kubernetes クラスタでの動的検証
- **kubectl プラグイン**: リソース作成前の事前検証
- **CI/CD 統合**: マニフェストファイルの自動検証
- **IDE 統合**: YAML 編集時のリアルタイム検証
