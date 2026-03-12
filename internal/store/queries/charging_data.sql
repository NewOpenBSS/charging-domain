-- name: GetChargingDataByChargeId :one
select charging_id,
       sequence_number,
       modified_on,
       charge_data
from charging_data
where charging_id = $1;

-- name: UpdateChargeData :exec
UPDATE charging_data
set sequence_number = $2,
    charge_data = $3,
    modified_on=now()
where charging_id = $1;


-- name: CreateChargeData :exec
insert into charging_data(charging_id, sequence_number, charge_data)
values ($1, $2, $3);

-- name: DeleteChargeDate :exec
delete
from charging_data
where charging_id = $1;


--
