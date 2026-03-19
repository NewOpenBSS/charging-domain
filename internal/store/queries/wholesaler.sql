-- name: AllWholesalers :many
-- Returns all active wholesaler records. Used by the tenant resolver to build
-- the hostname → wholesale_id lookup map.
SELECT id, hosts FROM wholesaler WHERE active = true;
