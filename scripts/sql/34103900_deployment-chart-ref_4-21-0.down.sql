DELETE FROM global_strategy_metadata_chart_ref_mapping WHERE chart_ref_id=(select id from chart_ref where version='4.21.0' and name='Deployment');

DELETE FROM "public"."chart_ref" WHERE ("location" = 'deployment-chart_4-21-0' AND "version" = '4.21.0');

UPDATE "public"."chart_ref" SET "is_default" = 't' WHERE "location" = 'deployment-chart_4-20-0' AND "version" = '4.20.0';