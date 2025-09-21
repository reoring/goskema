#!/bin/bash

# User API サンプルテストスクリプト

set -e

BASE_URL="http://localhost:8080"

echo "🧪 goskema User API Sample テスト開始"
echo "📍 サーバーが $BASE_URL で動作していることを確認してください"
echo

# Health check
echo "1️⃣ ヘルスチェック"
curl -s "$BASE_URL/health" | jq '.' 2>/dev/null || curl -s "$BASE_URL/health"
echo -e "\n"

# Get initial users
echo "2️⃣ 初期ユーザー一覧を取得"
curl -s "$BASE_URL/users" | jq '.' 2>/dev/null || curl -s "$BASE_URL/users"
echo -e "\n"

# Create a new user
echo "3️⃣ 新しいユーザーを作成"
curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "三郎",
    "email": "saburo@example.com",
    "age": 28,
    "active": true
  }' | jq '.' 2>/dev/null || curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "三郎",
    "email": "saburo@example.com",
    "age": 28,
    "active": true
  }'
echo -e "\n"

# Get specific user
echo "4️⃣ 特定ユーザー（ID=1）を取得"
curl -s "$BASE_URL/users/1" | jq '.' 2>/dev/null || curl -s "$BASE_URL/users/1"
echo -e "\n"

# Partial update using PATCH (demonstrating Presence)
echo "5️⃣ 部分更新（PATCH）でPresence機能をテスト（nameのみ変更）"
curl -X PATCH "$BASE_URL/users/1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "太郎改"
  }' | jq '.' 2>/dev/null || curl -X PATCH "$BASE_URL/users/1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "太郎改"
  }'
echo -e "\n"

# Check the updated user
echo "6️⃣ 更新後のユーザーを確認"
curl -s "$BASE_URL/users/1" | jq '.' 2>/dev/null || curl -s "$BASE_URL/users/1"
echo -e "\n"

# Test validation error
echo "7️⃣ バリデーションエラーをテスト（不正なデータ）"
curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "",
    "email": "invalid-email",
    "age": -5
  }' | jq '.' 2>/dev/null || curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "",
    "email": "invalid-email",
    "age": -5
  }'
echo -e "\n"

# Test unknown fields (should be rejected)
echo "8️⃣ 未知フィールドのテスト（UnknownStrictによりエラーになるはず）"
curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "四郎",
    "email": "shiro@example.com",
    "age": 22,
    "active": true,
    "unknown_field": "this should be rejected"
  }' | jq '.' 2>/dev/null || curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "四郎",
    "email": "shiro@example.com",
    "age": 22,
    "active": true,
    "unknown_field": "this should be rejected"
  }'
echo -e "\n"

# Get JSON Schema
echo "9️⃣ JSON Schema を取得"
curl -s "$BASE_URL/schema" | jq '.' 2>/dev/null || curl -s "$BASE_URL/schema"
echo -e "\n"

# Final user list
echo "🔟 最終的なユーザー一覧"
curl -s "$BASE_URL/users" | jq '.' 2>/dev/null || curl -s "$BASE_URL/users"
echo -e "\n"

echo "✅ テスト完了！"
echo
echo "🎯 学習ポイント："
echo "   - DSL でのスキーマ定義"
echo "   - Presence を活用した部分更新（PATCH）"
echo "   - 構造化されたエラーレスポンス"
echo "   - JSON Schema の自動生成"
echo "   - UnknownStrict による厳密な検証"
