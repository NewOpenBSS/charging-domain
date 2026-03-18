# Makefile for go-ocs project

# Variables
BINARY_DRA := charging-dra
BINARY_ENGINE := charging-engine
BINARY_BACKEND := charging-backend
CMD_DRA := ./cmd/charging-dra
CMD_ENGINE := ./cmd/charging-engine
CMD_BACKEND := ./cmd/charging-backend
DB_URL := "postgres://gobss:gobss@localhost:5432/gobss?sslmode=disable"
DB_SCHEMA := "&search_path=charging"
MIGRATIONS_PATH := db/migrations
SEEDS_PATH := db/seeds

# Default target
.PHONY: all
all: build

# Build targets
.PHONY: build
build: build-dra build-engine build-backend

.PHONY: build-dra
build-dra:
	go build -o $(BINARY_DRA) $(CMD_DRA)

.PHONY: build-engine
build-engine:
	go build -o $(BINARY_ENGINE) $(CMD_ENGINE)

.PHONY: build-backend
build-backend:
	go build -o $(BINARY_BACKEND) $(CMD_BACKEND)

# Database migration targets
.PHONY: migrate-up
migrate-up:
	migrate -verbose -path $(MIGRATIONS_PATH) -database $(DB_URL) up

.PHONY: migrate-down
migrate-down:
	migrate -verbose -path $(MIGRATIONS_PATH) -database $(DB_URL)$(DB_SCHEMA) down

.PHONY: migrate-clean
migrate-clean:
	migrate -verbose -database $(DB_URL) down drop
	migrate -verbose -database $(DB_URL) down -all -f

# Seeding targets
.PHONY: seed
seed:
	migrate -path $(SEEDS_PATH) -database $(DB_URL)$(DB_SCHEMA) up

# Clean targets
.PHONY: clean
clean:
	rm -f $(BINARY_DRA) $(BINARY_ENGINE) $(BINARY_BACKEND)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          Build all the applications"
	@echo "  build-dra      Build charging-dra"
	@echo "  build-engine   Build charging-engine"
	@echo "  build-backend  Build charging-backend"
	@echo "  migrate-up    Apply all migrations"
	@echo "  migrate-down  Rollback the last migration"
	@echo "  migrate-clean Clean the database"
	@echo "  seed          Seed the database"
	@echo "  clean         Remove built binaries"
	@echo "  help          Show this help"
