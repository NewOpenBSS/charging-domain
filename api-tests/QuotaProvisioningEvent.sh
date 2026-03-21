#!/usr/bin/env bash
# QuotaProvisioningEvent Kafka producer
# Topic: public.quota-provisioning-event
# Run from IntelliJ using the green play button, or from terminal: bash api-tests/QuotaProvisioningEvent.sh
#
# Prerequisites:
#   brew install kcat
#
# Environment: set KAFKA_BROKERS to override default
BROKER="${KAFKA_BROKERS:-localhost:9092}"
TOPIC="public.quota-provisioning-event"

produce() {
  local description="$1"
  local payload="$2"
  echo ">>> ${description}"
  echo "${payload}" | kcat -P -b "${BROKER}" -t "${TOPIC}" -H "event-type=${description}"
  echo "    produced to ${TOPIC}"
  echo ""
}

# ── CREATE ───────────────────────────────────────────────────────────────────

produce "quota.provisioned" '{
  "eventType": "quota.provisioned",
  "msisdn": "27831234567",
  "quotaId": "quota-001",
  "ratingGroup": 10,
  "totalVolume": 1073741824,
  "usedVolume": 0,
  "validFrom": "2026-03-21T00:00:00Z",
  "validTo": "2026-04-21T00:00:00Z"
}'

# ── UPDATE ───────────────────────────────────────────────────────────────────

produce "quota.updated" '{
  "eventType": "quota.updated",
  "msisdn": "27831234567",
  "quotaId": "quota-001",
  "ratingGroup": 10,
  "totalVolume": 2147483648,
  "usedVolume": 536870912,
  "validFrom": "2026-03-21T00:00:00Z",
  "validTo": "2026-04-21T00:00:00Z"
}'

# ── EXPIRE ───────────────────────────────────────────────────────────────────

produce "quota.expired" '{
  "eventType": "quota.expired",
  "msisdn": "27831234567",
  "quotaId": "quota-001",
  "ratingGroup": 10,
  "expiredAt": "2026-04-21T00:00:00Z"
}'

echo "Done."
