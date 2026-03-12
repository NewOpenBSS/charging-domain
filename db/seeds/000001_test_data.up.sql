--
-- Test MVNO
--
INSERT INTO charging.wholesaler (id, modified_on, active, legal_name, display_name, realm, hosts, nchfurl, ratelimit, rateplan_id, contract_id)
VALUES ('b0db9f5f-1195-4ef5-a9e3-b84662dd9cdd', '2025-04-22 21:33:17.572789', false, 'Inactive Test MVNO (Pty) Ltd', 'Inactive Test MVNO', 'test1',
        '{test1}',
        'http://localhost:8103/api',
        '0', '29c5e0b3-b7d3-44dd-9b53-e15c9303833f', '29c5e0b3-b7d3-44dd-9b53-e15c9303833f');
INSERT INTO charging.wholesaler (id, modified_on, active, legal_name, display_name, realm, hosts, nchfurl, ratelimit, rateplan_id, contract_id)
VALUES ('29b00b6f-3340-41d5-90eb-e4cb3321d511', '2025-04-22 21:17:14.136512', true, 'Test MVNO (Pty) Ltd', 'Test MVNO', 'test', '{test}',
        'http://localhost:8103/api', '0',
        '29c5e0b3-b7d3-44dd-9b53-e15c9303833f', '29c5e0b3-b7d3-44dd-9b53-e15c9303833f');

INSERT INTO charging.wholesaler (id, modified_on, active, legal_name, display_name, realm, hosts, nchfurl, ratelimit, rateplan_id, contract_id)
VALUES ('4cbc6ce8-f350-4b74-b8d4-e3380fc99494', '2025-04-22 21:17:14.136512', true, 'SdB MVNO (Pty) Ltd', 'SdB MVNO', 'sdebie', '{sdebie.localhost}',
        'http://localhost:8103/api', '0',
        '29c5e0b3-b7d3-44dd-9b53-e15c9303833f', '29c5e0b3-b7d3-44dd-9b53-e15c9303833f');
--
-- Test Subscribers
--
INSERT INTO charging.subscriber (subscriber_id, modified_on, status, rateplan_id, customer_id, wholesale_id, msisdn, iccid, contract_id)
VALUES ('cb2fec7d-1035-40f1-8d62-d3fbddfb6dc2', '2025-04-22 21:28:55.070749', 'ACTIVE', '492aded4-22a4-4d37-9939-022bb4171cfb',
        'c31a7ad9-205c-444d-b6d2-cf6cad697432', '29b00b6f-3340-41d5-90eb-e4cb3321d511', '0272587305',
        'E457A928-291C-410F-BD00-4F228967F49E', 'e457a928-291c-410f-bd00-4f228967f49e');
INSERT INTO charging.subscriber (subscriber_id, modified_on, status, rateplan_id, customer_id, wholesale_id, msisdn, iccid, contract_id)
VALUES ('d16ecd8e-3fb2-4886-b2bd-52d70c7a868c', '2025-04-22 21:32:08.793349', 'SUSPENDED', '492aded4-22a4-4d37-9939-022bb4171cfb',
        'c31a7ad9-205c-444d-b6d2-cf6cad697432', '336f5734-33e9-45c9-aaf4-4227614faeac', '0828939447',
        'D16ECD8E-3FB2-4886-B2BD-52D70C7A868C', 'd16ecd8e-3fb2-4886-b2bd-52d70c7a868c');
INSERT INTO charging.subscriber (subscriber_id, modified_on, status, rateplan_id, customer_id, wholesale_id, msisdn, iccid, contract_id)
VALUES ('70e8348d-0727-4093-9e29-a8ef46d8eecb', '2025-04-22 21:39:24.730444', 'SUSPENDED', '492aded4-22a4-4d37-9939-022bb4171cfb',
        '68bb1bac-580f-4618-a49b-186db4dfffd5', '68bb1bac-580f-4618-a49b-186db4dfffd5', '0828938000',
        '68bb1bac-580f-4618-a49b-186db4dfffd5', '68bb1bac-580f-4618-a49b-186db4dfffd5');

INSERT INTO charging.subscriber (subscriber_id, modified_on, status, rateplan_id, customer_id, wholesale_id, msisdn, iccid, contract_id)
VALUES ('e439cec6-a092-465e-89b7-d986a536ea90', '2025-04-22 21:39:24.730444', 'ACTIVE', '492aded4-22a4-4d37-9939-022bb4171cfb',
        '2e6664dd-92c9-4abb-96de-4282b869071f', '4cbc6ce8-f350-4b74-b8d4-e3380fc99494', '0271234567',
        'a8139f92-12c8-4029-9e5d-9548ed979494', 'd8bdb0cd-f613-4017-9fc2-564203f769f6');

--
-- Test Classification Plan
--
INSERT INTO charging.classification (classification_id, name, created_on, effective_time, created_by, approved_by, status, plan)
VALUES ('34c3f749-3608-4562-a53b-4132619cf952', 'Test Classification Plan', '2025-04-22 21:06:20.440663', '2025-04-23 00:00:00.000000', 'eddie',
        'eddie', 'ACTIVE', '{
    "ruleSetId": "34c3f749-3608-4562-a53b-4132619cf952",
    "ruleSetName": "Default Classification Rules",
    "useServiceWindows": true,
    "defaultServiceWindow": "STANDARD",
    "defaultSourceType": "HOME",
    "serviceWindows": {
      "PEAK": {
        "startTime": "08:00",
        "endTime": "20:00"
      },
      "OFFPEAK": {
        "startTime": "20:01",
        "endTime": "07:59"
      },
      "NIGHT_SAVER": {
        "startTime": "00:00",
        "endTime": "05:00"
      },
      "FREE_PERIOD": {
        "startTime": "00:00",
        "endTime": "23:59"
      }
    },
    "serviceTypes": [
      {
        "type": "VOICE",
        "chargingInformation": "IMS",
        "description": "Voice services including MO, MT, and MF calls",
        "sourceType": "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc, Req.NfConsumerIdentification.NfPLMNID.Mnc)",
        "serviceDirection": "serviceDirection(Info.RoleOfNode)",
        "serviceCategory": "serviceCategory(Info.CalledPartyAddress)",
        "unitType": "SECONDS",
        "serviceWindows": [
          "OFFPEAK",
          "PEAK"
        ]
      },
      {
        "type": "SMS",
        "chargingInformation": "SMS",
        "description": "Short Message Service",
        "sourceType": "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc,Req.NfConsumerIdentification.NfPLMNID.Mnc)",
        "serviceDirection": "''MO''",
        "serviceCategory": "Info.RecipientInfo[0].RecipientOtherAddress.SmAddressData)",
        "defaultServiceCategory": "UNKNOWN",
        "unitType": "UNITS",
        "serviceWindows": [
          "OFFPEAK",
          "PEAK"
        ]
      },
      {
        "type": "DATA",
        "chargingInformation": "PDU",
        "description": "Mobile data usage",
        "sourceType": "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc, Req.NfConsumerIdentification.NfPLMNID.Mnc)",
        "serviceDirection": "''MO''",
        "defaultServiceCategory": "INTERNET",
        "unitType": "OCTETS",
        "serviceWindows": [
          "OFFPEAK",
          "NIGHT_SAVER",
          "PEAK"
        ],
        "serviceCategory": "serviceCategory(Unit.RatingGroup)"
      },
      {
        "type": "USSD1",
        "chargingInformation": "NEF",
        "serviceTypeRule": "OneTimeEvent",
        "description": "Unstructured Supplementary Service Data",
        "sourceType": "sourceByMccMnc(NfConsumerIdentification.NfPLMNID.Mcc, NfConsumerIdentification.NfPLMNID.Mnc)",
        "serviceDirection": "''MO''",
        "defaultServiceCategory": "GENERAL",
        "unitType": "UNITS",
        "serviceWindows": [
          "BUSINESS_HOURS"
        ],
        "serviceCategory": "serviceCategory(Info.ExternalGroupIdentifier)"
      },
      {
        "type": "USSD2",
        "chargingInformation": "NEF",
        "serviceTypeRule": "not OneTimeEvent",
        "description": "Unstructured Supplementary Service Data",
        "sourceType": "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc, Req.NfConsumerIdentification.NfPLMNID.Mnc)",
        "serviceDirection": "''MO''",
        "defaultServiceCategory": "GENERAL",
        "unitType": "SECONDS",
        "serviceWindows": [
          "BUSINESS_HOURS"
        ],
        "serviceCategory": "serviceCategory(Info.ExternalGroupIdentifier)"
      }
    ]
  }');


--
-- Rate plans (Settlement, wholesale and retail)
--
INSERT INTO charging.rateplan (id, plan_id, modified_at, plan_type, wholesale_id, plan_name, rateplan, plan_status, created_by, approved_by,
                               effective_at)
VALUES (3, '9a0d171b-ab6d-45d9-ae2e-ddb281936855', '2025-04-22 21:19:02.264894', 'SETTLEMENT', null, 'Settlement Plan', '{
  "rateLines": [
    {
      "baseTariff": 0.12,
      "multiplier": 1,
      "tariffType": "ACTUAL",
      "description": "Data",
      "minimumUnits": 1,
      "unitOfMeasure": "1Mb",
      "classificationKey": "DATA.HOME.MO.*.*",
      "roundingIncrement": 1,
      "groupKey": "DATA-INTERNET"
    },
    {
      "baseTariff": 0.01,
      "multiplier": 1.5,
      "tariffType": "ACTUAL",
      "description": "Local Voice calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.NATIONAL.*",
      "roundingIncrement": 1,
      "groupKey": "NATIONAL-CALLS"
    },
    {
      "baseTariff": 0.05,
      "multiplier": 1,
      "qosProfile": "SILVER",
      "tariffType": "ACTUAL",
      "description": "Africa Voice calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.AFRICA.*",
      "roundingIncrement": 1,
      "groupKey": "AFRICA-CALLS"
    },
    {
      "baseTariff": 0.02,
      "multiplier": 2,
      "tariffType": "ACTUAL",
      "description": "International calls",
      "minimumUnits": 1,
      "monetaryOnly": true,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.ROAMING.*.INTERNATIONAL.*",
      "roundingIncrement": 1,
      "groupKey": "ROAMING-CALLS"
    },
    {
      "baseTariff": 0.001,
      "multiplier": 1,
      "tariffType": "ACTUAL",
      "description": "Text Messages",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "SMS.*.MO.*.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.001,
      "multiplier": 0.8,
      "tariffType": "ACTUAL",
      "description": "Data social",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "DATA.*.*.SOCIAL.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0,
      "multiplier": 1,
      "tariffType": "ACTUAL",
      "description": "Emergency calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.EMERGENCY.SOS",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.01,
      "multiplier": 1,
      "qosProfile": "NONE",
      "tariffType": "ACTUAL",
      "description": "Ussd calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "USSD2.HOME.MO.GENERAL.STANDARD",
      "roundingIncrement": 1
    }
  ],
  "ratePlanId": "439a3b7a-1bc4-4908-91bf-9e57b6b3fb5d",
  "ratePlanName": "Settlement Plan",
  "ratePlanType": "SETTLEMENT"
}', 'ACTIVE', 'eddie', 'eddie', '2025-04-23 00:00:00.000000');

INSERT INTO charging.rateplan (id, plan_id, modified_at, plan_type, wholesale_id, plan_name, rateplan, plan_status, created_by, approved_by,
                               effective_at)
VALUES (4, '29c5e0b3-b7d3-44dd-9b53-e15c9303833f', '2025-04-22 21:22:36.375897', 'WHOLESALE', '336f5734-33e9-45c9-aaf4-4227614faeac',
        'Wholesale Plan', '{
    "rateLines": [
      {
        "baseTariff": 0.18,
        "multiplier": 2,
        "tariffType": "ACTUAL",
        "description": "Data",
        "minimumUnits": 1,
        "unitOfMeasure": "1Mb",
        "classificationKey": "DATA.HOME.MO.*.*",
        "roundingIncrement": "5Kb",
        "groupKey": "DATA1"
      },
      {
        "baseTariff": 0.02,
        "multiplier": 1.5,
        "tariffType": "ACTUAL",
        "description": "Local Voice calls",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "VOICE.HOME.MO.NATIONAL.*",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0.05,
        "multiplier": 1,
        "qosProfile": "SILVER",
        "tariffType": "ACTUAL",
        "description": "Africa Voice calls",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "VOICE.HOME.MO.AFRICA.*",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0.03,
        "multiplier": 2,
        "tariffType": "ACTUAL",
        "description": "International calls",
        "minimumUnits": 1,
        "monetaryOnly": true,
        "unitOfMeasure": 1,
        "classificationKey": "VOICE.ROAMING.*.INTERNATIONAL.*",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0.005,
        "multiplier": 1,
        "tariffType": "ACTUAL",
        "description": "Text Messages",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "SMS.*.MO.*.*",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0.005,
        "multiplier": 0.8,
        "tariffType": "ACTUAL",
        "description": "Data social",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "DATA.*.*.SOCIAL.*",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0,
        "multiplier": 1,
        "tariffType": "ACTUAL",
        "description": "Emergency calls",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "VOICE.HOME.MO.EMERGENCY.SOS",
        "roundingIncrement": 1
      },
      {
        "baseTariff": 0.01,
        "multiplier": 1,
        "qosProfile": "NONE",
        "tariffType": "ACTUAL",
        "description": "Ussd calls",
        "minimumUnits": 1,
        "unitOfMeasure": 1,
        "classificationKey": "USSD2.HOME.MO.GENERAL.STANDARD",
        "roundingIncrement": 1
      }
    ],
    "ratePlanId": "29c5e0b3-b7d3-44dd-9b53-e15c9303833f",
    "ratePlanName": "Wholesale Plan",
    "ratePlanType": "WHOLESALE"
  }', 'ACTIVE', 'eddie', 'eddie', '2025-04-23 00:00:00.000000');

INSERT INTO charging.rateplan (id, plan_id, modified_at, plan_type, wholesale_id, plan_name, rateplan, plan_status, created_by, approved_by,
                               effective_at)
VALUES (5, '492aded4-22a4-4d37-9939-022bb4171cfb', '2025-04-22 21:24:03.683779', 'RETAIL', '29b00b6f-3340-41d5-90eb-e4cb3321d511', 'Retail Plan', '{
  "rateLines": [
    {
      "baseTariff": 0.25,
      "multiplier": 1,
      "tariffType": "ACTUAL",
      "description": "Default Internet",
      "minimumUnits": "15Kb",
      "unitOfMeasure": "1Mb",
      "classificationKey": "DATA.HOME.MO.*.*",
      "roundingIncrement": "5Kb",
      "groupKey": "DATA"
    },
    {
      "baseTariff": 0.0,
      "multiplier": 0,
      "tariffType": "ACTUAL",
      "description": "FREE Data",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "DATA.HOME.MO.FREE.*",
      "roundingIncrement": 1,
      "groupKey": "DATA"
    },
    {
      "baseTariff": 0.25,
      "multiplier": 2,
      "tariffType": "ACTUAL",
      "description": "Premium data",
      "minimumUnits": "20Kb",
      "unitOfMeasure": "1Mb",
      "classificationKey": "DATA.HOME.MO.PREMIUM.*",
      "roundingIncrement": "10Kb",
      "groupKey": "DATA"
    },
    {
      "baseTariff": 0.25,
      "multiplier": 1.3,
      "tariffType": "ACTUAL",
      "description": "Whatsapp Voice",
      "minimumUnits": "1Kb",
      "unitOfMeasure": "1Mb",
      "classificationKey": "DATA.HOME.MO.WHATSAPP.*",
      "roundingIncrement": "1Kb",
      "groupKey": "DATA"
    },
    {
      "baseTariff": 0.25,
      "multiplier": 1.5,
      "qosProfile": "SILVER",
      "tariffType": "ACTUAL",
      "description": "Local Voice calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.NATIONAL.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.05,
      "multiplier": 1,
      "qosProfile": "SILVER",
      "tariffType": "ACTUAL",
      "description": "Africa Voice calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.AFRICA.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.05,
      "multiplier": 2,
      "qosProfile": "BRONZE",
      "tariffType": "ACTUAL",
      "description": "International roaming calls",
      "minimumUnits": 1,
      "monetaryOnly": true,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.ROAMING.*.INTERNATIONAL.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.005,
      "multiplier": 1,
      "qosProfile": "STANDARD",
      "tariffType": "ACTUAL",
      "description": "Text Messages",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "SMS.*.MO.*.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.003,
      "multiplier": 0.8,
      "qosProfile": "GOLD",
      "tariffType": "ACTUAL",
      "description": "Data social",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "DATA.*.*.SOCIAL.*",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0,
      "multiplier": 1,
      "qosProfile": "NONE",
      "tariffType": "ACTUAL",
      "description": "Emergency calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "VOICE.HOME.MO.EMERGENCY.SOS",
      "roundingIncrement": 1
    },
    {
      "baseTariff": 0.01,
      "multiplier": 1,
      "qosProfile": "NONE",
      "tariffType": "ACTUAL",
      "description": "Ussd calls",
      "minimumUnits": 1,
      "unitOfMeasure": 1,
      "classificationKey": "USSD2.HOME.MO.GENERAL.STANDARD",
      "roundingIncrement": 1
    }
  ],
  "ratePlanId": "492aded4-22a4-4d37-9939-022bb4171cfb",
  "ratePlanName": "Premium Retail Plan",
  "ratePlanType": "RETAIL"
}', 'ACTIVE', 'eddie', 'eddie', '2025-04-23 00:00:00.000000');
