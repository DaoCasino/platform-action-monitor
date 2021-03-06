-- migrate:up
CREATE OR REPLACE FUNCTION chain.action_trace_notify_trigger() RETURNS trigger AS
$$
DECLARE
BEGIN
    PERFORM pg_notify('new_action_trace', NEW.receipt_global_sequence::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER action_trace_insert
    AFTER INSERT
    ON chain.action_trace
    FOR EACH ROW
EXECUTE PROCEDURE chain.action_trace_notify_trigger();

CREATE INDEX action_trace_index_name_sequence ON chain.action_trace USING btree (act_name, receipt_global_sequence asc);
-- migrate:down
DROP INDEX chain.action_trace_index_name_sequence;
DROP TRIGGER action_trace_insert ON chain.action_trace;
DROP FUNCTION chain.action_trace_notify_trigger;



