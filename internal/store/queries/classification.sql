-- name: FindActiveClassification :one
select classification_id,
       name,
       created_on,
       effective_time,
       created_by,
       approved_by,
       status,
       plan
from classification
where effective_time < now()
  and status = 'ACTIVE'
order by effective_time DESC
limit 1;
