-- name: DestinationGroupByGroupName :one
-- Retrieves a single destination group record by its group name.
SELECT group_name, region
FROM carrier_destination_group
WHERE group_name = $1;

-- name: CreateDestinationGroup :one
-- Inserts a new destination group and returns the full persisted row.
INSERT INTO carrier_destination_group (group_name, region)
VALUES ($1, $2)
RETURNING group_name, region;

-- name: UpdateDestinationGroup :one
-- Updates an existing destination group by group name and returns the updated row.
UPDATE carrier_destination_group
SET region = $2
WHERE group_name = $1
RETURNING group_name, region;

-- name: DeleteDestinationGroup :exec
-- Deletes a destination group by group name.
DELETE FROM carrier_destination_group
WHERE group_name = $1;
