CREATE UNIQUE INDEX checkin_user_date
ON checkin(user_id, year, month, day);
