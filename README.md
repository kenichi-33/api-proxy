# API Proxy

高性能で拡張性の高い Go 言語製 HTTP プロキシサーバー。

## 特徴

- **Clean Architecture**: テスト容易性と保守性を重視した設計。
- **設定管理**: YAML ファイルによる柔軟な設定（ホットリロード対応）。
- **ルーティング**: パスベースのバックエンドルーティング。
- **認証**: バックエンドごとの認証設定（現在は Basic 認証に対応）。
- **変換**: リクエスト/レスポンスのヘッダー操作やカスタム変換。
- **可観測性**:
    - 構造化ログ (JSON)
    - Request ID / Correlation ID の自動付与
    - リクエスト所要時間の計測 (`X-Backend-Duration`, `total_duration_ms`)
- **管理機能**: ヘルスチェックや設定リロード用の管理 API。
- **CI/CD**: GitHub Actions による Lint, Test, Build の自動化。

## 必要要件

- Go 1.23 以上

## インストールと実行

### 1. ビルド

```bash
go build -o proxy cmd/proxy/main.go
```

### 2. 設定ファイルの準備

以下の2つの設定ファイルが必要です。

- `server.yaml`: サーバー設定（ポートなど）
- `backends.yaml`: バックエンド設定（ルーティング、認証など）

**server.yaml 例:**
```yaml
port: 8080
admin_port: 9095
read_timeout: 10s
write_timeout: 10s
idle_timeout: 60s
```

**backends.yaml 例:**
```yaml
backends:
  - name: "example-backend"
    locations:
      - path: "/api/v1"
        backend_url: "http://localhost:8081"
        timeout: 5s
        methods: ["GET", "POST"]
    auth:
      type: "basic"
      config:
        username: "admin"
        password: "password"
```

### 3. 実行

```bash
./proxy --server-config server.yaml --backend-config backends.yaml
```

開発時は以下のように実行できます：

```bash
go run cmd/proxy/main.go --server-config server.yaml --backend-config backends.yaml
```

## API 利用例

```bash
curl -v -u admin:password http://localhost:8080/api/v1/test
```

## 管理 API

デフォルトではポート 9095 で管理 API が提供されます。

- ヘルスチェック: `GET /health`
- 設定リロード: `POST /reload`

## 開発

### テスト実行

```bash
go test -race -v ./...
```

### Lint 実行

```bash
golangci-lint run
```

## ライセンス

MIT
