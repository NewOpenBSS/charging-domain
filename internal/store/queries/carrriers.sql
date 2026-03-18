-- name: AllCarriers :many
select *
from carrier
order by plmn;

-- name: CarrierByPlmn :one
-- Retrieves a single carrier record by its PLMN identifier.
SELECT plmn, modified_on, mcc, mnc, carrier_name, source_group,
       destination_group, country_name, iso
FROM carrier
WHERE plmn = $1;

-- name: CreateCarrier :one
-- Inserts a new carrier and returns the full persisted row including modified_on.
INSERT INTO carrier (
    plmn, mcc, mnc, carrier_name, source_group,
    destination_group, country_name, iso, modified_on
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
) RETURNING plmn, modified_on, mcc, mnc, carrier_name, source_group,
            destination_group, country_name, iso;

-- name: UpdateCarrier :one
-- Updates an existing carrier by PLMN and returns the updated row.
-- modified_on is refreshed to NOW() on every update.
UPDATE carrier
SET mcc               = $2,
    mnc               = $3,
    carrier_name      = $4,
    source_group      = $5,
    destination_group = $6,
    country_name      = $7,
    iso               = $8,
    modified_on       = NOW()
WHERE plmn = $1
RETURNING plmn, modified_on, mcc, mnc, carrier_name, source_group,
          destination_group, country_name, iso;

-- name: DeleteCarrier :exec
-- Deletes a carrier by PLMN.
DELETE FROM carrier
WHERE plmn = $1;
