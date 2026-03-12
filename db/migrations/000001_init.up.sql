--
-- Extensions needed
--
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS charging;
SET search_path TO charging;

-- Initial DB Structure
create table wholesaler
(
    id           uuid primary key,
    modified_on  TIMESTAMPTZ default now() not null,
    active       boolean     default false not null,
    legal_name   varchar                   not null,
    display_name varchar                   not null,
    realm        varchar                   not null,
    hosts        text[]                    not null,
    nchfUrl      varchar                   not null,
    rateLimit    numeric     default 0     not null,
    contract_id  uuid                      not null,
    rateplan_id  uuid                      not null
);

comment on table wholesaler is 'The Wholesaler shadow database';
comment on column wholesaler.modified_on is 'The last time the record was updated';
comment on column wholesaler.active is 'True if the wholesaler is enabled for trading';
comment on column wholesaler.legal_name is 'The name of the wholesaler';
comment on column wholesaler.display_name is 'The display name of the wholesaler';
comment on column wholesaler.realm is 'The realm for the wholesaler';
comment on column wholesaler.hosts is 'A comma delimited list of valid hosts';
comment on column wholesaler.nchfUrl is 'The NCHF Url for the wholesaler';
comment on column wholesaler.rateLimit is 'The ratelimit for NCHF requests';
comment on column wholesaler.rateplan_id is 'The rate plan to use for this wholesaler';
comment on column wholesaler.contract_id is 'The contract for the wholesaler';

--
-- Subscriber
--
create table subscriber
(
    subscriber_id      uuid                               not null
        constraint subscriber_pk
            primary key,
    modified_on        TIMESTAMPTZ default now()          not null,
    rateplan_id        uuid                               not null,
    customer_id        uuid                               not null,
    wholesale_id       uuid                               not null,
    msisdn             varchar                            not null,
    iccid              varchar                            not null,
    contract_id        uuid                               not null,
    status             text        default 'ACTIVE'::text not null,
    allow_oob_charging boolean     default true           not null
);

comment on table subscriber is 'The shadow subscriber table';
comment on column subscriber.subscriber_id is 'The subscriber id as allocated by the OSS service';
comment on column subscriber.modified_on is 'The last time the record was modified';
comment on column subscriber.status is 'The status of the subscriber: ACTIVE, INACTIVE, BLOCKED';
comment on column subscriber.rateplan_id is 'The rate plan id for the subscriber';
comment on column subscriber.customer_id is 'The Customer ID linked to the Subscriber ID';
comment on column subscriber.wholesale_id is 'The wholesaler Id the subscriber is provisioned on';
comment on column subscriber.msisdn is 'The MSISDN linked to the subscriber';
comment on column subscriber.iccid is 'The ICCID linked to the subscriber';
comment on column subscriber.contract_id is 'The contract id linked to the msisdn';
comment on column subscriber.allow_oob_charging is 'The allow out of bundle charging flag';

--
-- Carrier
--
create table carrier
(
    plmn              varchar
        constraint carrier_pk
            primary key,
    modified_on       TIMESTAMPTZ default now() not null,
    mcc               varchar(3)                not null,
    mnc               varchar(3),
    carrier_name      varchar                   not null,
    source_group      varchar                   not null,
    destination_group varchar                   not null,
    country_name      varchar                   not null,
    iso               varchar                   not null
);

comment on table carrier is 'The table of all carriers the network interacts with';

comment on column carrier.plmn is 'The PLMN for the carrier';

comment on column carrier.modified_on is 'The last time the record was updated';

comment on column carrier.mcc is 'The Mobile Country Code';

comment on column carrier.mnc is 'The Mobile Network Code';

comment on column carrier.carrier_name is 'The carrier name';

comment on column carrier.source_group is 'The roaming source group for the carrier';

comment on column carrier.destination_group is 'The destination group for the carrier';

comment on column carrier.country_name is 'The country name';

comment on column carrier.iso is 'The ISO country code';

create unique index carrier_mcc_mnc_uindex
    on carrier (mcc, mnc);

--
-- Carrier Source Group
--
create table carrier_source_group
(
    group_name varchar not null
        constraint carrier_source_group_pk
            primary key,
    region     varchar not null
);

comment on table carrier_source_group is 'Valid roaming source options';

comment on column carrier_source_group.group_name is 'The valid group name';

comment on column carrier_source_group.region is 'The full region name';

--
-- Carrier Destination Group
--
create table carrier_destination_group
(
    group_name varchar not null
        constraint carrier_destination_group_pk
            primary key,
    region     varchar not null
);

comment on table carrier_destination_group is 'The destination groups';

comment on column carrier_destination_group.group_name is 'The valid destination group name';

comment on column carrier_destination_group.region is 'The full region name';

--
-- Number Plan
--
create table number_plan
(
    number_id     bigserial
        constraint number_plan_pk
            primary key,
    modified_on   TIMESTAMPTZ default now() not null,
    name          varchar                   not null,
    plmn          varchar                   not null,
    number_range  varchar                   not null,
    number_length integer     default 99    not null
);

comment on table number_plan is 'The number plans for the carriers';

comment on column number_plan.modified_on is 'The last time the record was updated';

comment on column number_plan.name is 'The name of the number plan';

comment on column number_plan.plmn is 'The plmn of carrier the numbers belong too';

comment on column number_plan.number_range is 'The first portion of the number';

comment on column number_plan.number_length is 'The maximum length a number can be.';

--
-- Classification
--
create table classification
(
    classification_id uuid                                         not null
        constraint classification_pk
            primary key,
    name              varchar                                      not null,
    created_on        timestamp default now()                      not null,
    effective_time    TIMESTAMPTZ                                  not null,
    created_by        varchar                                      not null,
    approved_by       varchar,
    status            varchar   default 'DRAFT'::character varying not null,
    Plan              jsonb                                        not null
);

comment on table classification is 'The classification store';

comment on column classification.name is 'The name of the classification rules';

comment on column classification.created_on is 'The time the record was created';

comment on column classification.effective_time is 'The time from which the plan is effective';

comment on column classification.created_by is 'The user that created the classification plan';

comment on column classification.approved_by is 'The user that approved the plan';

comment on column classification.status is 'The status of the plan: DRAFT, PENDING, ACTIVE and RETRIED';

comment on column classification.plan is 'The classification plan';

--
-- Charging Data
--
create table charging_data
(
    charging_id     varchar                   not null
        constraint charging_data_pk
            primary key,
    sequence_number bigint      default 0     not null,
    modified_on     TIMESTAMPTZ default now() not null,
    charge_data     jsonb                     not null
);

comment on column charging_data.sequence_number is 'The sequence number link to the request';

comment on table charging_data is 'The charging data session information';

comment on column charging_data.charging_id is 'The session reference Id';

comment on column charging_data.modified_on is 'The last time the record was modified';

comment on column charging_data.charge_data is 'The JSON structure of the charging data';


--
-- Charging trace table
--
create table charging_trace
(
    trace_id       uuid    default public.uuid_generate_v4() not null
        constraint pk_charging_trace
            primary key,
    created_at     TIMESTAMPTZ                               not null,
    request        jsonb                                     not null,
    response       jsonb,
    execution_time bigint                                    not null,
    charging_id    varchar                                   not null,
    sequence_nr    integer default 0                         not null,
    msisdn         varchar                                   not null
);

comment on table charging_trace is 'Trace table to store charging request and respons ';

comment on column charging_trace.trace_id is 'Unique id for the trace';

comment on column charging_trace.created_at is 'The timestamp of when the entry was made';

comment on column charging_trace.request is 'The JSONB request object';

comment on column charging_trace.response is 'The JSONB response object';

comment on column charging_trace.execution_time is 'The execution time in milliseconds';

comment on column charging_trace.charging_id is 'The charging id assigned by the NF';

comment on column charging_trace.sequence_nr is 'The sequence number as assigned by the NF';

comment on column charging_trace.msisdn is 'The MSISDN of the subscriber';

create index charging_trace_created_at_index
    on charging_trace (created_at);


--
-- RatePlan Table
--
create table rateplan
(
    id           bigserial primary key,
    plan_id      uuid                                           not null,
    modified_at  TIMESTAMPTZ default now()                      not null,
    plan_type    varchar                                        not null,
    wholesale_id uuid,
    plan_name    varchar                                        not null,
    rateplan     jsonb                                          not null,
    plan_status  varchar     default 'DRAFT'::character varying not null,
    created_by   varchar                                        not null,
    approved_by  varchar,
    effective_at TIMESTAMPTZ                                    not null
);

comment on table rateplan is 'The rateplan for all';

comment on column rateplan.id is 'The unique key for the plan';

comment on column rateplan.plan_id is 'The plan id';

comment on column rateplan.plan_name is 'The rate plan name';

comment on column rateplan.plan_type is 'The type of plan SETTLEMENT, WHOLESALE, RETAIL';

comment on column rateplan.wholesale_id is 'The wholesaler the rate plan belongs to. Can be null if it is a SETTLEMENT plan or WHOLESALE plan';

comment on column rateplan.rateplan is 'The YAML rate plan';

comment on column rateplan.created_by is 'The user that created the rate plan';

comment on column rateplan.approved_by is 'The user that approved the rate plan';

comment on column rateplan.modified_at is 'The last time the rate plan was change';

comment on column rateplan.effective_at is 'The time the rate plan become effective';

comment on column rateplan.plan_status is 'The status of the rule plan: DRAFT, PENDING_APPROVAL, ACTIVE, RETIRED';

--
-- Quota Table
--
create table quota
(
    quota_id         uuid primary key default public.uuid_generate_v4(),
    last_modified    TIMESTAMPTZ not null default now(),
    subscriber_id    uuid        not null,
    next_action_time TIMESTAMPTZ,
    quota            jsonb       not null
);

comment on table quota is 'Stores a JSON-based quota counter structure per subscriber';

comment on column quota.quota_id is 'Primary key for the quota record';
comment on column quota.last_modified is 'Timestamp used for optimistic locking and tracking updates';
comment on column quota.subscriber_id is 'The subscriber ID this quota is assigned to';
comment on column quota.next_action_time is 'The next time the quota should be evaluated (e.g., for expiry or renewal)';
comment on column quota.quota is 'The quota counter details stored as a JSONB structure';
