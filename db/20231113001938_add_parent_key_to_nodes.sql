ALTER TABLE nodes ADD parent_key VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE nodes ADD level INT NOT NULL DEFAULT 0;