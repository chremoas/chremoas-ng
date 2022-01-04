CREATE OR REPLACE FUNCTION getMemberRoles(BIGINT, BOOL, BOOL) RETURNS SETOF roles AS
$$
DECLARE
    inputUserID ALIAS FOR $1;
    inputSig ALIAS FOR $2;
    inputSync ALIAS FOR $3;
    _role  roles%ROWTYPE;
    exists bigint;
BEGIN
    FOR _role IN SELECT * FROM roles WHERE sync = inputSync AND sig = inputSig
        LOOP
            SELECT INTO exists user_id
            FROM filter_membership
            WHERE filter in (SELECT filter FROM role_filters WHERE role = _role.id)
              AND user_id = inputUserID
            GROUP BY user_id
            HAVING count(*) = (SELECT count(*) FROM role_filters WHERE role = _role.id);

            IF exists > 0 THEN
                RETURN NEXT _role;
            END IF;
        END LOOP;
END;
$$ LANGUAGE plpgsql;