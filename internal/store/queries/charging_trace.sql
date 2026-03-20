-- name: FindChargingTraceByIdSeqNr :one
select trace_id,
       created_at,
       request,
       response,
       execution_time,
       charging_id,
       sequence_nr,
       msisdn

from charging_trace
where charging_id = $1
  and sequence_nr = $2;


-- name: CreateChargingTrace :one
insert into charging_trace(trace_id, charging_id, sequence_nr, created_at, request, response, execution_time, msisdn)
values (gen_random_uuid(), $1, $2, now(), $3, $4, $5, $6)
returning trace_id;

-- name: FindChargingTraceByTraceId :one
SELECT trace_id,
       created_at,
       request,
       response,
       execution_time,
       charging_id,
       sequence_nr,
       msisdn
FROM charging_trace
WHERE trace_id = $1;
