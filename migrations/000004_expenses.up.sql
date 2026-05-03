CREATE TABLE expenses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    title text NOT NULL,
    description text NULL,
    total_amount_minor bigint NOT NULL,
    currency text NOT NULL DEFAULT 'THB',
    paid_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    split_type text NOT NULL,
    receipt_url text NULL,
    expense_date date NULL,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT expenses_title_length CHECK (length(title) BETWEEN 1 AND 160),
    CONSTRAINT expenses_total_amount_positive CHECK (total_amount_minor > 0),
    CONSTRAINT expenses_currency_valid CHECK (currency = 'THB'),
    CONSTRAINT expenses_split_type_valid CHECK (split_type IN ('equal', 'manual'))
);

CREATE TRIGGER expenses_set_updated_at
BEFORE UPDATE ON expenses
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE expense_participants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id uuid NOT NULL REFERENCES expenses(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    share_amount_minor bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT expense_participants_expense_user_unique UNIQUE (expense_id, user_id),
    CONSTRAINT expense_participants_share_positive CHECK (share_amount_minor > 0)
);

CREATE TABLE debts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id uuid NOT NULL REFERENCES expenses(id) ON DELETE RESTRICT,
    debtor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    creditor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    original_amount_minor bigint NOT NULL,
    remaining_amount_minor bigint NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    accepted_at timestamptz NULL,
    rejected_at timestamptz NULL,
    settled_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT debts_expense_debtor_creditor_unique UNIQUE (expense_id, debtor_id, creditor_id),
    CONSTRAINT debts_no_self_debt CHECK (debtor_id <> creditor_id),
    CONSTRAINT debts_original_amount_positive CHECK (original_amount_minor > 0),
    CONSTRAINT debts_remaining_amount_non_negative CHECK (remaining_amount_minor >= 0),
    CONSTRAINT debts_remaining_amount_within_original CHECK (remaining_amount_minor <= original_amount_minor),
    CONSTRAINT debts_status_valid CHECK (status IN ('pending', 'accepted', 'rejected', 'partially_settled', 'settled'))
);

CREATE TRIGGER debts_set_updated_at
BEFORE UPDATE ON debts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_expenses_group_date ON expenses(group_id, expense_date DESC, created_at DESC);
CREATE INDEX idx_expense_participants_user_id ON expense_participants(user_id);
CREATE INDEX idx_debts_debtor_status ON debts(debtor_id, status);
CREATE INDEX idx_debts_creditor_status ON debts(creditor_id, status);
CREATE INDEX idx_debts_expense_id ON debts(expense_id);
