# goskema サンプルプロジェクト集

このディレクトリには、goskema の機能を実際に体験できるサンプルプロジェクトが含まれています。

## プロジェクト一覧

### 1. [user-api](./user-api/) - User管理RESTful API 🚀
**難易度**: ⭐⭐☆☆☆ (初級)

- **主な特徴**: DSLでのスキーマ定義、Presenceを活用したPATCH操作、構造化エラーハンドリング
- **学べること**: 基本的なCRUD操作、部分更新の実装、エラーレスポンス
- **実行時間**: 約5分

```bash
cd user-api
go mod tidy
go run .  # サーバー起動
# 別ターミナルで: ./test.sh
```

**ハイライト**: PATCH操作でPresence情報を使って「本当に更新されたフィールド」だけを変更

### 2. [k8s-crd-validator](./k8s-crd-validator/) - Kubernetes CRD検証 ☸️
**難易度**: ⭐⭐⭐☆☆ (中級)

- **主な特徴**: CRDスキーマの読み込み、AdmissionWebhook風の検証、YAML/JSON対応
- **学べること**: Kubernetesとの連携、YAML処理、厳密な検証
- **実行時間**: 約10分

```bash
cd k8s-crd-validator  
go mod tidy
go run . demo  # 全機能デモ
# または: ./test.sh
```

**ハイライト**: 実際のKubernetes CRDからgoskemaスキーマを生成して厳密検証

### 3. [config-manager](./config-manager/) - 設定ファイル管理システム ⚙️
**難易度**: ⭐⭐⭐⭐☆ (上級)

- **主な特徴**: 複数フォーマット対応、環境変数置換、階層設定、ストリーミング処理
- **学べること**: 設定管理、大きなファイルの処理、環境別設定、バリデーション
- **実行時間**: 約15分

```bash
cd config-manager
go mod tidy
go run . generate --template  # テンプレート生成
go run . demo  # または: ./test.sh
```

**ハイライト**: 環境変数展開 (`${VAR:-default}`) と階層的設定継承

## 実行方法

各プロジェクトディレクトリに移動して：

```bash
# 1. 依存関係のセットアップ
go mod init <sample-name>
go mod edit -replace github.com/reoring/goskema=../..
go mod tidy

# 2. 実行
go run .

# 3. または、テストスクリプト
./test.sh
```

## 推奨学習順序

1. **user-api** から始めて goskema の基本を理解
2. **k8s-crd-validator** で実用的な検証パターンを学習  
3. **config-manager** で高度な設定管理パターンを体験

## 共通の学習ポイント

### 🎯 核心機能

1. **型安全なスキーマ定義**: DSLを使った直感的なスキーマ記述
   ```go
   schema := g.ObjectOf[User]().
       Field("email", g.StringOf[string]()).Required().
       UnknownStrict().MustBind()
   ```

2. **Presenceの活用**: 欠落・null・defaultの区別
   ```go
   if decoded.Presence["/name"] & goskema.PresenceSeen != 0 {
       // フィールドが実際に送信された
   }
   ```

3. **エラーハンドリング**: JSON Pointerでの正確なエラー位置
   ```go
   if issues, ok := goskema.AsIssues(err); ok {
       for _, issue := range issues {
           fmt.Printf("Error at %s: %s\n", issue.Path, issue.Message)
       }
   }
   ```

4. **ストリーミング処理**: 大きなデータの効率的な処理
   ```go
   result, err := goskema.ParseFrom(ctx, schema, goskema.JSONReader(r))
   ```

### 🔧 実用パターン

- **RESTful API**: リクエスト検証とレスポンス生成
- **設定管理**: 環境別設定と変数展開
- **Kubernetes連携**: CRD検証とWebhook実装
- **相互運用**: JSON Schema/OpenAPI生成

### 🚀 パフォーマンス

- **メモリ効率**: ストリーミング処理で大きなJSONも安心
- **エラー収集**: Fail-fast vs Collect モードの使い分け
- **JSON Schema**: 既存ツールとの互換性

## サンプル実行結果

### user-api の例
```bash
$ curl -X PATCH localhost:8080/users/1 -d '{"name":"次郎"}'
{
  "user": {"id":1,"name":"次郎","email":"taro@example.com",...},
  "updated_fields": ["name"]
}
```

### k8s-crd-validator の例
```bash
$ go run . validate invalid-widget.yaml
❌ Validation failed with 3 issue(s):
  1. 🚨 required field missing at /spec/size
  2. 🚨 value too large at /spec/replicas
  3. 🚨 unknown property at /spec/config/unknown_field
```

### config-manager の例
```bash
$ DB_PASSWORD=secret go run . show --env=production
app:
  environment: production
  port: 80
database:
  password: "***masked***"
```

## 応用例

これらのサンプルを参考に以下が作成可能：

- **API Gateway**: 外部APIとの連携とスキーマ検証
- **AdmissionWebhook**: Kubernetesクラスタでの動的検証  
- **設定管理**: マイクロサービスの環境別設定
- **データパイプライン**: ETL処理での型安全な変換
- **CLI ツール**: 設定ファイルやマニフェストの検証

## トラブルシューティング

### よくある問題

1. **`go mod tidy` でエラー**
   ```bash
   go mod edit -replace github.com/reoring/goskema=../..
   ```

2. **ポート使用中エラー (user-api)**  
   ```bash
   # 別のポートを使用するか、既存プロセスを停止
   lsof -ti:8080 | xargs kill
   ```

3. **環境変数が展開されない (config-manager)**
   ```bash
   # 明示的に環境変数を設定
   export DB_PASSWORD=your_password
   ```

## フィードバック

各サンプルは独立して動作し、段階的に複雑さが増すように設計されています。
問題やご提案があれば、ぜひ Issue でお知らせください！

---

**Happy Coding with goskema! 🎉**