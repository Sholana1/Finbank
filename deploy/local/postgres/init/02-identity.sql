CREATE ROLE identity_owner LOGIN PASSWORD 'identity_owner_local';
CREATE ROLE identity_app LOGIN PASSWORD 'identity_app_local';

CREATE DATABASE identity OWNER identity_owner;

REVOKE ALL ON DATABASE identity FROM PUBLIC;
GRANT CONNECT ON DATABASE identity TO identity_app