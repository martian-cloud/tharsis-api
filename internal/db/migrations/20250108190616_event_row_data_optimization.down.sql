CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $$
DECLARE row RECORD;
notification json;

    BEGIN -- Convert the old or new row to JSON, based on the kind of action.
-- Action = DELETE?             -> OLD row
-- Action = INSERT or UPDATE?   -> NEW row
IF (TG_OP = 'DELETE') THEN row = OLD;
ELSE row = NEW;
END IF;

        -- Construct the notification as a JSON string.
notification = json_build_object(
    'table',
    TG_TABLE_NAME,
    'action',
    TG_OP,
    'id',
    row.id
);

        -- Execute pg_notify(channel, notification)
PERFORM pg_notify('events', notification::text);

        -- Result is ignored since this is an AFTER trigger
RETURN NULL;
END;

$$ LANGUAGE plpgsql;
