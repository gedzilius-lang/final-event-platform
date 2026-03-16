-- migrations/orders/002_add_guest_user_id.sql
-- Adds guest_user_id to orders.orders.
-- Set at finalization time to record which guest's balance was debited.
-- Required for compensating events on void: we need the guest's user_id to credit back.

SET search_path = orders;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS guest_user_id uuid;

COMMENT ON COLUMN orders.guest_user_id IS
  'user_id of the guest whose NC balance was debited. Set at finalization. NULL for pending/voided-before-paid orders.';
