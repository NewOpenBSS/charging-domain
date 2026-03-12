-- name: AllNumbers :many
select number_plan.*, carrier.destination_group
from number_plan
inner join carrier on number_plan.plmn = carrier.plmn;
