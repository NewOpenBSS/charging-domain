#!/usr/bin/env bash
# SubscriberEvent Kafka producer
# Topic: public.subscriber-event
# Run from IntelliJ using the green play button, or from terminal: bash api-tests/SubscriberEvent.sh
#
# Prerequisites:
#   brew install kcat
#
# Environment: set KAFKA_BROKERS to override default
BROKER="${KAFKA_BROKERS:-localhost:9092}"
TOPIC="public.subscriber-event"

produce() {
  local description="$1"
  local payload="$2"
  echo ">>> ${description}"
  echo "${payload}" | kcat -P -b "${BROKER}" -t "${TOPIC}" -H "event-type=${description}"
  echo "    produced to ${TOPIC}"
  echo ""
}

# ── CREATE ──────────────────────────────────────────────────────────────────

produce "subscriber.created" '{
  "eventType": "subscriber.created",
  "msisdn": "27831234567",
  "subscriberId": "sub-001",
  "firstName": "John",
  "lastName": "Doe",
  "status": "ACTIVE",
  "subscriberType": "PREPAID",
  "createdAt": "2026-03-21T10:00:00Z"
}'

# ── UPDATE ───────────────────────────────────────────────────────────────────

produce "subscriber.updated" '{
  "eventType": "subscriber.updated",
  "msisdn": "27831234567",
  "subscriberId": "sub-001",
  "firstName": "John",
  "lastName": "Doe",
  "status": "SUSPENDED",
  "subscriberType": "PREPAID",
  "updatedAt": "2026-03-21T11:00:00Z"
}'

# ── DELETE ───────────────────────────────────────────────────────────────────

produce "subscriber.deleted" '{
  "eventType": "subscriber.deleted",
  "msisdn": "27831234567",
  "subscriberId": "sub-001",
  "deletedAt": "2026-03-21T12:00:00Z"
}'

echo "Done."
