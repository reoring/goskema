# User管理RESTful API サンプル

goskemaの基本機能を活用したUser管理APIサーバーのサンプルです。

## 特徴

- **DSLスキーマ定義**: 直感的なUser構造体のスキーマ定義
- **Presence活用**: PATCH操作で欠落・null・defaultを正確に区別
- **構造化エラー**: JSON Pointerでの正確なエラー位置表示
- **JSON Schema出力**: 定義したスキーマをJSON Schemaとして出力可能

## API エンドポイント

### GET /users
全ユーザーを取得

### GET /users/{id}
特定のユーザーを取得

### POST /users
新しいユーザーを作成
```json
{
  "name": "太郎",
  "email": "taro@example.com",
  "age": 30,
  "active": true
}
```

### PATCH /users/{id}
ユーザーを部分更新（Presenceを活用）
```json
{
  "name": "次郎"
  // emailやageが指定されていない場合は更新しない
  // nullが指定された場合は明示的にnullに設定
}
```

### DELETE /users/{id}
ユーザーを削除

### GET /schema
ユーザースキーマをJSON Schema形式で出力

## 実行方法

```bash
# 依存関係のセットアップ
go mod init user-api-sample
go mod edit -replace github.com/reoring/goskema=../..
go mod tidy

# サーバー起動
go run .
```

サーバーは http://localhost:8080 で起動します。

## テスト例

```bash
# ユーザー作成
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"太郎","email":"taro@example.com","age":30,"active":true}'

# ユーザー取得
curl http://localhost:8080/users/1

# 部分更新（nameのみ変更）
curl -X PATCH http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"次郎"}'

# スキーマ取得
curl http://localhost:8080/schema
```

## 学習ポイント

1. **スキーマ定義**: `g.ObjectOf[User]()` を使った型安全なスキーマ
2. **バリデーション**: メールフォーマット、年齢範囲などの制約
3. **Presence追跡**: PATCH操作での部分更新の実現
4. **エラーハンドリング**: 構造化されたエラーレスポンス
5. **JSON Schema出力**: API仕様書の自動生成
