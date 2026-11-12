-- +goose Up
ALTER TABLE wfm.pause_template ALTER COLUMN created_by DROP NOT NULL, ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE wfm.shift_template ALTER COLUMN created_by DROP NOT NULL, ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE wfm.working_condition ALTER COLUMN created_by DROP NOT NULL, ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE wfm.agent_absence ALTER COLUMN created_by DROP NOT NULL, ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE wfm.forecast_calculation ALTER COLUMN created_by DROP NOT NULL, ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE wfm.agent_working_conditions ALTER COLUMN updated_by DROP NOT NULL;

-- +goose Down
ALTER TABLE wfm.agent_working_conditions ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE wfm.forecast_calculation ALTER COLUMN created_by SET NOT NULL, ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE wfm.agent_absence ALTER COLUMN created_by SET NOT NULL, ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE wfm.working_condition ALTER COLUMN created_by SET NOT NULL, ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE wfm.shift_template ALTER COLUMN created_by SET NOT NULL, ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE wfm.pause_template ALTER COLUMN created_by SET NOT NULL, ALTER COLUMN updated_by SET NOT NULL;
