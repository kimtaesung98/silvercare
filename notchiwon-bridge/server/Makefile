.PHONY: run test vet fmt generate generate-go generate-dart migrate-up migrate-down

# 코드 생성 도구 버전 (CI와 같게 유지)
SQLC_VERSION := v1.30.0
OAPI_CODEGEN_VERSION := v2.5.1
GOOSE_VERSION := v3.26.0
OPENAPI_GENERATOR_VERSION := 7.26.0

TOOLS_DIR := .tools
OPENAPI_GENERATOR_JAR := $(TOOLS_DIR)/openapi-generator-cli-$(OPENAPI_GENERATOR_VERSION).jar
DART_CLIENT_DIR := ../app/packages/api_client/lib/src/generated

run:
	go run ./cmd/api

# Postgres가 필요한 테스트는 TEST_DATABASE_URL이 있을 때만 돕니다.
test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# 계약(openapi.yaml, 마이그레이션, 쿼리)을 바꾸면 실행하고 생성 결과를 같은 PR에 커밋합니다.
generate: generate-go generate-dart

generate-go:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) -config api/oapi-codegen.yaml api/openapi.yaml

$(OPENAPI_GENERATOR_JAR):
	mkdir -p $(TOOLS_DIR)
	curl -fsSL --retry 5 --retry-all-errors -o $@.tmp \
		https://repo.maven.apache.org/maven2/org/openapitools/openapi-generator-cli/$(OPENAPI_GENERATOR_VERSION)/openapi-generator-cli-$(OPENAPI_GENERATOR_VERSION).jar
	mv $@.tmp $@

generate-dart: $(OPENAPI_GENERATOR_JAR)
	rm -rf $(TOOLS_DIR)/dart-out $(DART_CLIENT_DIR)
	java -jar $(OPENAPI_GENERATOR_JAR) generate -g dart -i api/openapi.yaml -o $(TOOLS_DIR)/dart-out \
		--additional-properties=pubName=api_client >/dev/null
	mkdir -p $(DART_CLIENT_DIR)
	cp -r $(TOOLS_DIR)/dart-out/lib/. $(DART_CLIENT_DIR)/
	dart format $(DART_CLIENT_DIR) >/dev/null

# 로컬 DB에 마이그레이션 적용 (DATABASE_URL 사용). 단계 2부터는 서버 시작 시 자동 적용합니다.
migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir db/migrations postgres "$(DATABASE_URL)" down
