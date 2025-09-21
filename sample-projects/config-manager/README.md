# Config Manager サンプル

goskema を使った高機能な設定ファイル管理システムのサンプルです。

## 特徴

- **複数フォーマット対応**: JSON, YAML 両対応
- **環境変数置換**: `${ENV_VAR}` 形式の環境変数展開
- **階層設定**: 開発/ステージング/本番環境の設定継承
- **ストリーミング処理**: 大きな設定ファイルの効率的な処理
- **厳密なバリデーション**: 型安全な設定値の検証
- **ホットリロード**: 設定変更の動的反映

## 設定構造

このサンプルでは、Web アプリケーションの設定を管理します：

```go
type Config struct {
    App        AppConfig        `json:"app"`
    Database   DatabaseConfig   `json:"database"`
    Redis      RedisConfig      `json:"redis"`
    Logging    LoggingConfig    `json:"logging"`
    Features   FeaturesConfig   `json:"features"`
}

type AppConfig struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Environment string            `json:"environment"`
    Port        int               `json:"port"`
    Host        string            `json:"host"`
    TLS         TLSConfig        `json:"tls"`
    Cors        CorsConfig       `json:"cors"`
    Metadata    map[string]string `json:"metadata"`
}

type DatabaseConfig struct {
    Host         string `json:"host"`
    Port         int    `json:"port"`
    Database     string `json:"database"`
    Username     string `json:"username"`
    Password     string `json:"password"`
    MaxConns     int    `json:"maxConns"`
    MaxIdleConns int    `json:"maxIdleConns"`
    SSLMode      string `json:"sslMode"`
}
```

## 環境別設定の継承

1. **base.yaml** - 基本設定
2. **development.yaml** - 開発環境の差分
3. **staging.yaml** - ステージング環境の差分
4. **production.yaml** - 本番環境の差分

設定は base → 環境別 の順で継承・上書きされます。

## 実行方法

```bash
# 依存関係のセットアップ
go mod init config-manager-sample
go mod edit -replace github.com/reoring/goskema=../..
go mod tidy

# 設定検証
go run . validate --env=development
go run . validate --env=production

# 設定表示（マスク付き）
go run . show --env=development

# 設定ファイル生成
go run . generate --template

# ホットリロード（設定変更監視）
go run . watch --env=development
```

## サンプル設定ファイル

### base.yaml
```yaml
app:
  name: "MyWebApp"
  version: "1.0.0"
  host: "0.0.0.0"
  port: 8080
  tls:
    enabled: false
  cors:
    enabled: true
    origins: ["*"]

database:
  host: "localhost"
  port: 5432
  database: "myapp"
  username: "postgres"
  maxConns: 10
  maxIdleConns: 5
  sslMode: "prefer"

redis:
  host: "localhost"
  port: 6379
  database: 0
  poolSize: 10

logging:
  level: "info"
  format: "json"
  output: "stdout"

features:
  analytics: true
  debugging: false
```

### development.yaml
```yaml
app:
  environment: "development"
  port: 3000

database:
  password: "${DB_PASSWORD:-dev_password}"
  sslMode: "disable"

redis:
  password: "${REDIS_PASSWORD:-}"

logging:
  level: "debug"
  output: "stdout"

features:
  debugging: true
```

### production.yaml
```yaml
app:
  environment: "production"
  port: 80
  tls:
    enabled: true
    certFile: "${TLS_CERT_FILE}"
    keyFile: "${TLS_KEY_FILE}"
  cors:
    origins: ["https://example.com", "https://app.example.com"]

database:
  host: "${DB_HOST}"
  password: "${DB_PASSWORD}"
  maxConns: 50
  maxIdleConns: 10
  sslMode: "require"

redis:
  host: "${REDIS_HOST}"
  password: "${REDIS_PASSWORD}"
  poolSize: 50

logging:
  level: "warn"
  output: "${LOG_OUTPUT:-stdout}"

features:
  analytics: true
  debugging: false
```

## テスト例

```bash
# 開発環境設定の検証
DB_PASSWORD=devpass go run . validate --env=development

# 本番環境設定の検証（必要な環境変数をセット）
DB_HOST=db.example.com \
DB_PASSWORD=secretpass \
REDIS_HOST=redis.example.com \
REDIS_PASSWORD=redispass \
TLS_CERT_FILE=/path/to/cert.pem \
TLS_KEY_FILE=/path/to/key.pem \
go run . validate --env=production

# 設定値の確認（パスワードはマスク）
go run . show --env=development

# ストリーミングでの大きな設定ファイル処理
go run . validate --streaming large-config.yaml
```

## 学習ポイント

1. **階層的設定管理**: 基本設定 + 環境差分の継承
2. **環境変数展開**: `${VAR}` や `${VAR:-default}` 形式の処理
3. **ストリーミング処理**: 大きな設定ファイルの効率的な読み込み
4. **設定値マスキング**: パスワード等の機密情報の表示制御  
5. **動的バリデーション**: 環境に応じた検証ルールの適用
6. **ホットリロード**: ファイル変更の検知と再読み込み

## 応用例

- **Kubernetes ConfigMap/Secret 管理**: YAML ファイルから K8s リソース生成
- **Docker Compose 設定**: 環境別の docker-compose.yml 生成
- **CI/CD パイプライン**: 環境別デプロイ設定の管理
- **マイクロサービス設定**: 複数サービスの設定一元管理
