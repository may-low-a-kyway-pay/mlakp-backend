CREATE TABLE payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    debt_id uuid NOT NULL REFERENCES debts(id) ON DELETE RESTRICT,
    paid_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    received_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_minor bigint NOT NULL,
    status text NOT NULL DEFAULT 'pending_confirmation',
    note text NULL,
    confirmed_at timestamptz NULL,
    rejected_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payments_no_self_payment CHECK (paid_by <> received_by),
    CONSTRAINT payments_amount_positive CHECK (amount_minor > 0),
    CONSTRAINT payments_status_valid CHECK (status IN ('pending_confirmation', 'confirmed', 'rejected'))
);

CREATE TRIGGER payments_set_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_payments_debt_status ON payments(debt_id, status);
CREATE INDEX idx_payments_paid_by ON payments(paid_by);
CREATE INDEX idx_payments_received_by ON payments(received_by);
