-- Add per-device user mini-program control permission.
ALTER TABLE `devices`
    ADD COLUMN `user_control_enabled` TINYINT(1) NOT NULL DEFAULT 0;
