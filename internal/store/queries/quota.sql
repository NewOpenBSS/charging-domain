-- name: FindQuota :one
select *
from quota
where subscriber_id = $1;

-- name: UpdateQuota :execrows
update quota
set last_modified=$4,
    quota=$3
where quota_id = $1
  and last_modified = $2;

-- name: CreateQuota :one
insert into quota(quota_id, last_modified, subscriber_id, next_action_time, quota)
VALUES ($1, now(), $2, now(), $3)
returning *;

-- name: FindExpiredQuotaSubscribers :many
-- Returns the subscriber_id for every quota row whose next_action_time is in the past
-- relative to the given reference time. Used by the housekeeping job to find dormant
-- subscribers with expired counters.
SELECT subscriber_id
FROM quota
WHERE next_action_time < $1;
