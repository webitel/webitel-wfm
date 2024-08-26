-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA wfm;

CREATE OR REPLACE FUNCTION wfm.tg_populate_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now()::timestamp AT TIME ZONE 'UTC';
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE FUNCTION wfm.tg_obj_default_rbac() RETURNS TRIGGER
    SECURITY DEFINER
    LANGUAGE plpgsql
AS
$$
BEGIN
    EXECUTE format(
            'INSERT INTO %I.%I AS acl (dc, object, grantor, subject, access)
             SELECT $1, $2, rbac.grantor, rbac.subject, rbac.access
               FROM (
                -- NEW object OWNER access SUPER(255) mode (!)
                SELECT $3, $3, (255)::int2
                 UNION ALL
                SELECT DISTINCT ON (rbac.subject)
                  -- [WHO] grants MAX of WINDOW subset access level
                    first_value(rbac.grantor) OVER sub
                  -- [WHOM] role/user administrative unit
                  , rbac.subject
                  -- [GRANT] ALL of WINDOW subset access mode(s)
                  , bit_or(rbac.access) OVER sub

                  FROM directory.wbt_default_acl AS rbac
                  JOIN directory.wbt_class AS oc ON (oc.dc, oc.name) = ($1, %L)
                  -- EXISTS( OWNER membership WITH grantor role )
                  -- JOIN directory.wbt_auth_member AS sup ON (sup.role_id, sup.member_id) = (rbac.grantor, $3)
                 WHERE rbac.object = oc.id
                   AND rbac.subject <> $3
                    -- EXISTS( OWNER membership WITH grantor user/role )
                   AND (rbac.grantor = $3 OR EXISTS(SELECT true
                         FROM directory.wbt_auth_member sup
                        WHERE sup.member_id = $3
                          AND sup.role_id = rbac.grantor
                       ))
                WINDOW sub AS (PARTITION BY rbac.subject ORDER BY rbac.access DESC)

               ) AS rbac(grantor, subject, access)',
            tg_table_schema,
            tg_table_name || '_acl',
            tg_argv[0]::name -- objclass: directory.wbt_class.name
            )
        --      :srv,   :oid,   :rid
        USING NEW.domain_id, NEW.id, NEW.created_by;
    -- FOR EACH ROW
    RETURN NEW;

END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION wfm.tg_obj_default_rbac;

DROP FUNCTION wfm.tg_populate_updated_at_column;

DROP SCHEMA wfm CASCADE;
-- +goose StatementEnd
