BEGIN;
ALTER TABLE hook_definitions DROP hide_execution_metadata;
ALTER TABLE hook_schedules DROP hide_execution_metadata;
ALTER TABLE hook_executions DROP request_payload;
END;