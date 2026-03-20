-- name: InsertSubscriber :exec
-- Inserts a new subscriber row. modified_on is set to NOW() by the DB default.
INSERT INTO subscriber (subscriber_id, rateplan_id, customer_id, wholesale_id, msisdn, iccid, contract_id, status, allow_oob_charging)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateSubscriber :exec
-- Updates all mutable fields for an existing subscriber. modified_on is refreshed to NOW().
UPDATE subscriber
SET rateplan_id        = $2,
    customer_id        = $3,
    wholesale_id       = $4,
    msisdn             = $5,
    iccid              = $6,
    contract_id        = $7,
    status             = $8,
    allow_oob_charging = $9,
    modified_on        = NOW()
WHERE subscriber_id = $1;

-- name: DeleteSubscriber :exec
-- Hard-deletes a subscriber by subscriber_id.
DELETE FROM subscriber
WHERE subscriber_id = $1;
