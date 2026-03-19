# Project Briefing: go-ocs

## Project Purpose
The `go-ocs` project implements a charging platform for telecommunications billing systems, specifically designed for Online Charging System (OCS) functionality. It handles real-time charging, rating, quota management, and accounting for voice, data, and messaging services.

## Architecture Overview
This is a **domain-driven Go application** with a layered architecture:
- **Transport Layer**: HTTP API endpoints (using chi router)
- **Business Logic Layer**: Charging engine with step-based processing
- **Domain Layer**: Quota management, rating, and charge calculations
- **Persistence Layer**: SQL database access via pgx
- **Event Layer**: Kafka integration for event streaming

## Package Map

### Core Packages
- **`internal/chargeengine`**: Main charging logic, HTTP handlers, and processing pipeline
- **`internal/charging`**: Domain models (RateKey, UnitType) for charging data
- **`internal/quota`**: Quota management system with reservation, debit, and release operations
- **`internal/ruleevaluator`**: Policy rule evaluation engine
- **`internal/store`**: Database repository pattern implementation

### Infrastructure Packages  
- **`internal/appl`**: Application lifecycle management
- **`internal/baseconfig`**: Configuration loading
- **`internal/events`**: Kafka event producer
- **`internal/logging`**: Structured logging
- **`internal/nchf`**: 3GPP NCHF protocol models
- **`internal/diameter`**: Diameter protocol support

### Database
- **`db/migrations`**: SQL migration scripts (PostgreSQL)
- **`db/seeds`**: Test data seeding

## Key Components

### Charging Pipeline
The application processes charging requests through a step-based pipeline:
1. **Authenticate**: Validate subscriber and carrier
2. **Classify**: Determine service type and rating plan
3. **Rate**: Calculate costs and apply tariffs
4. **Account**: Debit quota and record transactions
5. **Process Charge Record**: Create audit trail

### Quota Management
Comprehensive quota system with:
- **Reservation**: Pre-allocate units with validity periods
- **Debit**: Consume used units with reclaim logic
- **Release**: Return unused quota
- **Tax calculation**: Built-in tax support

### Event Sourcing
- Kafka integration for charge records
- Quota journal events
- Notification events

## Entry Points

### Applications
1. **Charging Engine** (`cmd/charging-engine/main.go`):
   - HTTP server at `:8080`
   - NCHF charging API endpoints
   - Kafka producer for events

2. **DRA Server** (`cmd/charging-dra/main.go`):
   - Diameter protocol support
   - Rate limiting for wholesalers

## External Integrations

### Required Services
- **PostgreSQL**: Primary data store (connection string in config)
- **Kafka**: Event streaming (brokers configured in YAML)
- **Prometheus**: Metrics endpoint (`:9090/metrics`)

### Dependencies
Key Go dependencies from `go.mod`:
- `twmb/franz-go`: Kafka client
- `jackc/pgx/v5`: PostgreSQL driver
- `go-chi/chi/v5`: HTTP router
- `fiorix/go-diameter/v4`: Diameter protocol
- `prometheus/client_golang`: Metrics collection

## Risk Areas

### Complex Domain Logic
- **Rating Engine**: Complex rule evaluation for pricing
- **Quota Accounting**: Precise unit calculation and tax application
- **Charge Pipeline**: Multi-step workflow with retries

### Integration Points
- **Kafka reliability**: Event delivery guarantees
- **Database transactions**: Quota operations require ACID compliance
- **Protocol compliance**: NCHF and Diameter standards

## Configuration Structure
YAML-based configuration with:
- Database connection strings
- Kafka broker addresses
- HTTP server addresses
- Metrics endpoints
- Logging levels

## Development Notes
- Extensive test coverage (test files for all major components)
- SQL queries managed via sqlc
- Domain models follow 3GPP telecommunications standards (NCHF, Diameter)
- Metrics instrumentation throughout the codebase

This briefing provides the foundation for understanding the codebase and performing development tasks safely. The architecture follows clean separation of concerns with business logic centrally managed in the `internal` packages.