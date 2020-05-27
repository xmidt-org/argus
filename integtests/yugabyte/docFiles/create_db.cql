CREATE KEYSPACE IF NOT EXISTS argus;
CREATE TABLE argus.gifnoc (
    bucket VARCHAR,
    id VARCHAR,
    data blob,
    PRIMARY KEY (bucket, id))
    WITH default_time_to_live = 300
    AND transactions = {'enabled': 'false'};
