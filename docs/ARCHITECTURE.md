# Architecture

## System Overview
The go-ocs system is a telecommunications Online Charging System (OCS) that processes real-time charging requests, manages subscriber quotas, and handles accounting for voice, data, and messaging services. The system follows 3GPP standards (NCHF protocol) and supports multiple telecom interfaces including HTTP and Diameter.

## Major Components

### Charging Engine
Handles HTTP-based charging requests through a pipeline-based processing model. Main entry point at `:8080` serving NCHF-compliant APIs.

### DRA Server
Diameter protocol support for wholesale carrier rating and limiting. Entry point implements the Diameter Ro interface.

### Quota Management
Central quota reservation, debit, and release system with:
- Pre-allocation with validity periods
- Partial usage accounting
- Unused quota reclaim logic
- Tax calculation support

### Rating Engine
Policy-based pricing evaluation with:
- Rate plan selection
- Unit pricing calculation
- Tariff application

### Event Sourcing
Kafka-based event stream for:
- Charge record publishing
- Quota journaling
- Notification delivery

### Store/Persistence
PostgreSQL-backed repository pattern implementation with:
- SQL migration system
- Test data seeding
- Connection pooling

## Layering and Boundaries

### Transport Layer
- HTTP (chi router) endpoints in `internal/chargeengine`
- Diameter protocol handling in `internal/diameter`
- Prometheus metrics endpoints

### Service Layer
- Charging service orchestration (`internal/chargeengine-chargeservice.go`)
- Quota management service (`internal/quota/manager.go`)
- Business process steps in `internal/chargeengine/engine/steps/`

### Domain Layer
- Charging domain models (`internal/charging/`)
- Quota domain logic (`internal/quota/`)
- Rating models (`internal/chargeengine/model/`) 
- Rule evaluation (`internal/ruleevaluator/`)

### Persistence Layer
- SQL queries via sqlc (`internal/store/queries/`)
- Database repository interfaces (`internal/store/store.go`)
- Migration scripts in `db/migrations/`

### Messaging Layer
- Kafka producer/consumer (`internal/events/`)
- Charge event models
- Event serialization

### Configuration Layer
- YAML-based configuration loading (`internal/baseconfig/`)
- Environment-specific settings
- Secrets management

## Key Flows

### Charge Request Flow
1. Request received via HTTP (NCHF protocol)
2. Authentication: validate subscriber and carrier
3. Classification: determine service type and rate plan
4. Rating: calculate costs and apply pricing
5. Accounting: debit quota and record transaction
6. Response building and charge record creation
7. Kafka event published for audit trail

### Quota Update Flow
1. Quota reservation request with units and validity
2. Database transaction to pre-allocate quota
3. Kafka event published for quota journal
4. Subsequent debit on actual usage
5. Release unused quota when session ends

### Subscriber Lookup Flow
1. Extract MSISDN from charging request
2. Query database via `store.FindSubscriber()`
3. Validate subscriber status and balances
4. Load associated rate plans and tariffs

### Configuration Flow
1. Load YAML configuration at startup
2. Validate required fields
3. Initialize logging with config settings
4. Establish database connections
5. Start Kafka producers/consumers

## External Integrations

### Required Systems
- **PostgreSQL**: Primary data store with connection string configuration
- **Kafka**: Event streaming platform (multiple brokers supported)
- **Prometheus**: Metrics collection and monitoring

### Protocols
- **HTTP/REST**: Primary API protocol via chi router
- **Diameter**: Ro interface for wholesale carrier integration
- **NCHF**: 3GPP protocol for charging data requests

### Dependencies
- `franz-go`: Kafka client library
- `pgx/v5`: PostgreSQL driver with connection pooling
- `go-diameter/v4`: Diameter protocol implementation
- `prometheus/client_golang`: Metrics instrumentation

## Data and Control Boundaries

### Deterministic Components
- Rating calculations must be reproducible
- Quota debits must be atomic and transactional
- Charge record sequencing for audit trail integrity

### Isolation Boundaries
- Database transactions isolate quota operations
- Kafka events isolated per subscriber/charge
- HTTP endpoints isolated per charging session

### Control Flow Rules
- No business logic in transport layer handlers
- No domain queries in persistence layer
- Event publishing isolated from core charging logic
- Configuration isolated from runtime state

## Extension Points

### API Extensions
- Add new HTTP endpoints in `internal/chargeengine/`
- Extend NCHF protocol handlers
- Add Diameter applications to DRA server

### Service Extensions
- Add new charge processing steps in pipeline
- Extend quota management with new policies
- Add new rate calculation strategies

### Consumer Extensions
- Add Kafka topic consumers for new event types
- Extend event sourcing with audit hooks
- Add monitoring consumers

### Supporting Components
- Add new database repositories
- Extend configuration with new sections
- Add logging hooks and metrics

## Risk Areas

### Charging Risks
- Incorrect rate calculation leading to revenue loss
- Double charging on network retries
- Quota leaks from improper release

### Quota Mutation Risks
- Race conditions on concurrent debits
- Over-debiting due to expired grants
- Database transaction failures leaving inconsistent state

### Idempotency Risks
- Duplicate charge records on retry
- Duplicate Kafka events
- Duplicate quota reservations

### Replay Risks
- Network retransmission causing duplicate charges
- Kafka consumer replay processing duplicate events
- State corruption from replayed messages

### Concurrency Risks
- Multiple simultaneous debits on same quota
- Race conditions in rating engine
- Connection pool exhaustion under load

## Architectural Rules

1. Business logic must remain in domain layer (`internal/` packages)
2. Transport handlers must delegate to services
3. All database access through repository interfaces
4. Kafka events must be idempotent
5. Quota operations must be transactional
6. No direct file system access in business logic
7. All external calls must be configurable and mockable for tests
8. Metrics must be instrumented for all public endpoints
9. Logging must include correlation IDs for tracing
10. Configuration must be loadable without environment variables