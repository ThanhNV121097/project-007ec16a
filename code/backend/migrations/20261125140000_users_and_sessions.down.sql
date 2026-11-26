-- Local reset only: the server never runs a down file. Dropping the tables takes
-- their indexes, constraints and trigger with them; set_updated_at() stays, it
-- belongs to the helpers migration.
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
