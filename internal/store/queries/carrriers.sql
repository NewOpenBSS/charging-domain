-- name: AllCarriers :many
select *
from carrier
order by plmn;
