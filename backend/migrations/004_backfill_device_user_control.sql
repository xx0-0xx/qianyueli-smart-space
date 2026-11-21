-- Preserve the chosen default: door locks can be controlled by users; all other existing devices are opt-in.
UPDATE `devices`
SET `user_control_enabled` = CASE WHEN `type` = 'door_lock' THEN 1 ELSE 0 END;
