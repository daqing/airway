ALTER TABLE nodes ADD parent_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD level INT NOT NULL DEFAULT 0;
