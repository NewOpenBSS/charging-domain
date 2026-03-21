-- name: AllWholesalers :many
-- Returns all active wholesaler records. Used by the tenant resolver to build
-- the hostname → wholesale_id lookup map.
SELECT id, hosts FROM wholesaler WHERE active = true;

-- name: UpsertWholesaler :exec
-- Inserts a new wholesaler row or updates all mutable fields when the id already exists.
-- modified_on is refreshed to NOW() on both insert and update.
INSERT INTO wholesaler (id, modified_on, active, legal_name, display_name, realm, hosts, nchfUrl, rateLimit, contract_id, rateplan_id)
VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE
    SET modified_on  = NOW(),
        active       = EXCLUDED.active,
        legal_name   = EXCLUDED.legal_name,
        display_name = EXCLUDED.display_name,
        realm        = EXCLUDED.realm,
        hosts        = EXCLUDED.hosts,
        nchfUrl      = EXCLUDED.nchfUrl,
        rateLimit    = EXCLUDED.rateLimit,
        contract_id  = EXCLUDED.contract_id,
        rateplan_id  = EXCLUDED.rateplan_id;

-- name: SetWholesalerActive :exec
-- Sets the active flag on a wholesaler. Used for deregistering and suspend events.
UPDATE wholesaler
SET active = $2, modified_on = NOW()
WHERE id = $1;

-- name: DeleteWholesaler :exec
-- Hard-deletes a wholesaler by id.
DELETE FROM wholesaler
WHERE id = $1;

-- name: CountSubscribersByWholesaler :one
-- Counts the number of subscribers associated with a given wholesaler.
SELECT COUNT(*) FROM subscriber WHERE wholesale_id = $1;

-- name: DeleteInactiveWholesalerIfEmpty :exec
-- Atomically deletes a wholesaler only when it is inactive AND has no remaining subscribers.
-- This is a no-op when the wholesaler is still active or still has subscribers.
DELETE FROM wholesaler
WHERE id = $1
  AND active = false
  AND (SELECT COUNT(*) FROM subscriber WHERE wholesale_id = $1) = 0;
