-- name: AllNumbers :many
select number_plan.*, carrier.destination_group
from number_plan
inner join carrier on number_plan.plmn = carrier.plmn;

-- name: FindNumberPlanByID :one
-- Returns a single number plan row by its primary key.
SELECT number_id, modified_on, name, plmn, number_range, number_length
FROM number_plan
WHERE number_id = $1;

-- name: CreateNumberPlan :one
-- Inserts a new number plan row and returns it with the generated number_id and modified_on.
INSERT INTO number_plan (name, plmn, number_range, number_length, modified_on)
VALUES ($1, $2, $3, $4, NOW())
RETURNING number_id, modified_on, name, plmn, number_range, number_length;

-- name: UpdateNumberPlan :one
-- Updates all mutable fields of a number plan row and stamps modified_on.
UPDATE number_plan
SET name          = $2,
    plmn          = $3,
    number_range  = $4,
    number_length = $5,
    modified_on   = NOW()
WHERE number_id = $1
RETURNING number_id, modified_on, name, plmn, number_range, number_length;

-- name: DeleteNumberPlan :exec
-- Permanently deletes a number plan row by its primary key.
DELETE FROM number_plan WHERE number_id = $1;
