update carrier
set source_group='NATIONAL',
    destination_group='NATIONAL'
where country_name = 'New Zealand';

update carrier
set source_group='HOME',
    destination_group='NATIONAL'
where plmn = '53024';
