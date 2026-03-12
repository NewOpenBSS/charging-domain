-- name: FindSubscriberWithWholesalerByMSISDN :one
SELECT
    -- subscriber
    s.subscriber_id,
    s.modified_on,
    s.rateplan_id,
    s.customer_id,
    s.wholesale_id,
    s.msisdn,
    s.iccid,
    s.contract_id,
    s.status,
    s.allow_oob_charging,

    -- wholesaler (aliased to avoid name collisions)
    w.id           AS wholesaler_id,
    w.modified_on  AS wholesaler_modified_on,
    w.active       AS wholesaler_active,
    w.legal_name   AS wholesaler_legal_name,
    w.display_name AS wholesaler_display_name,
    w.realm        AS wholesaler_realm,
    w.hosts        AS wholesaler_hosts,
    w.nchfurl      AS wholesaler_nchfurl,
    w.ratelimit    AS wholesaler_ratelimit,
    w.contract_id  AS wholesaler_contract_id,
    w.rateplan_id  AS wholesaler_rateplan_id
FROM subscriber s
         JOIN wholesaler w ON w.id = s.wholesale_id
WHERE s.msisdn = $1;
