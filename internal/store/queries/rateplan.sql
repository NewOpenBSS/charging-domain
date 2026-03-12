-- name: FindActiveRatePlans :many
select *
from rateplan
where plan_status = 'ACTIVE'
  and effective_at < now() -- all plans that are effective now or in the past
order by plan_type,
         effective_at desc; -- order by effective_at descending so that the most recent plan is first
