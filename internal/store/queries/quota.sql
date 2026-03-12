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
