-- Stand-ins for the platform objects the migrations reference: only the keys
-- their foreign keys need, and cc_get_lookup as the platform defines it.
CREATE SCHEMA directory;
CREATE SCHEMA call_center;
CREATE SCHEMA flow;

CREATE TABLE directory.wbt_domain (dc BIGINT PRIMARY KEY, name TEXT);
CREATE TABLE directory.wbt_user (
    id BIGINT PRIMARY KEY, dc BIGINT NOT NULL REFERENCES directory.wbt_domain (dc),
    name TEXT, username TEXT, UNIQUE (dc, id)
);

CREATE TABLE call_center.cc_agent (id BIGINT PRIMARY KEY, domain_id BIGINT NOT NULL, user_id BIGINT NOT NULL, UNIQUE (domain_id, id));
CREATE TABLE call_center.cc_team (id BIGINT PRIMARY KEY, domain_id BIGINT NOT NULL, name TEXT, UNIQUE (domain_id, id));
CREATE TABLE call_center.cc_skill (id BIGINT PRIMARY KEY, domain_id BIGINT NOT NULL, name TEXT, UNIQUE (domain_id, id));
CREATE TABLE call_center.cc_pause_cause (id BIGINT PRIMARY KEY, domain_id BIGINT NOT NULL, name TEXT, UNIQUE (domain_id, id));

CREATE FUNCTION call_center.cc_get_lookup(_id BIGINT, _name CHARACTER VARYING) RETURNS JSONB
    LANGUAGE plpgsql IMMUTABLE AS
$$
BEGIN
    IF _id ISNULL THEN
        RETURN NULL;
    END IF;
    RETURN json_build_object('id', _id, 'name', _name)::jsonb;
END;
$$;

CREATE TABLE flow.calendar_timezones (id BIGINT PRIMARY KEY, sys_name TEXT NOT NULL);
CREATE TYPE flow.calendar_except_date AS (name TEXT, date BIGINT, repeat BOOLEAN, disabled BOOLEAN);
CREATE TABLE flow.calendar (
    id BIGINT PRIMARY KEY, domain_id BIGINT NOT NULL, name TEXT,
    timezone_id BIGINT REFERENCES flow.calendar_timezones (id),
    excepts flow.calendar_except_date[], UNIQUE (domain_id, id)
);
