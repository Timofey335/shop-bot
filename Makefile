include .env

LOCAL_BIN=$(CURDIR)/bin

LOCAL_MIGRATION_DIR=$(MIGRATION_DIR)
LOCAL_MIGRATION_DB="postgres://botuser:botpass@localhost:5432/shopbot?sslmode=disable"

install-deps:
	GOBIN=$(LOCAL_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

get-deps:
	go get github.com/jackc/pgx/v5/pgxpool
	go get github.com/google/uuid
	go get github.com/go-telegram-bot-api/telegram-bot-api/v5
	go get github.com/redis/go-redis/v9

migration-up:
	${LOCAL_BIN}/migrate -path ${MIGRATION_DIR} -database ${LOCAL_MIGRATION_DB} up

migration-down:
	${LOCAL_BIN}/migrate -path ${MIGRATION_DIR} -database ${LOCAL_MIGRATION_DB} down

migration-version:
	${LOCAL_BIN}/migrate -path ${MIGRATION_DIR} -database ${LOCAL_MIGRATION_DB} version