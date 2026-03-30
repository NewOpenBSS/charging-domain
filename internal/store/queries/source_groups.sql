-- name: SourceGroupByGroupName :one
-- Retrieves a single source group record by its group name.
SELECT group_name, region
FROM carrier_source_group
WHERE group_name = $1;

-- name: CreateSourceGroup :one
-- Inserts a new source group and returns the full persisted row.
INSERT INTO carrier_source_group (group_name, region)
VALUES ($1, $2)
RETURNING group_name, region;

-- name: UpdateSourceGroup :one
-- Updates an existing source group by group name and returns the updated row.
UPDATE carrier_source_group
SET region = $2
WHERE group_name = $1
RETURNING group_name, region;

-- name: DeleteSourceGroup :exec
-- Deletes a source group by group name.
DELETE FROM carrier_source_group
WHERE group_name = $1;
