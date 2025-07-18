BEGIN;
ALTER TABLE hook_definitions ADD hide_execution_metadata BOOLEAN DEFAULT false;
ALTER TABLE hook_schedules ADD hide_execution_metadata BOOLEAN DEFAULT false;
ALTER TABLE hook_executions ADD request_payload TEXT;
END;