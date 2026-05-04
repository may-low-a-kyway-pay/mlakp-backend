-- name: GetDebtForPaymentMark :one
SELECT d.id,
       d.debtor_id,
       d.creditor_id,
       d.remaining_amount_minor,
       d.status,
       COALESCE((
           SELECT SUM(p.amount_minor)
           FROM payments p
           WHERE p.debt_id = d.id
             AND p.status = 'pending_confirmation'
       ), 0)::bigint AS pending_amount_minor
FROM debts d
WHERE d.id = $1
FOR UPDATE;

-- name: CreatePayment :one
INSERT INTO payments (debt_id, paid_by, received_by, amount_minor, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, debt_id, paid_by, received_by, amount_minor, status, note, confirmed_at, rejected_at, created_at, updated_at;

-- name: GetPaymentWithDebtForUpdate :one
SELECT p.id,
       p.debt_id,
       p.paid_by,
       p.received_by,
       p.amount_minor,
       p.status,
       p.note,
       p.confirmed_at,
       p.rejected_at,
       p.created_at,
       p.updated_at,
       d.status AS debt_status,
       d.remaining_amount_minor AS debt_remaining_amount_minor
FROM payments p
JOIN debts d ON d.id = p.debt_id
WHERE p.id = $1
FOR UPDATE OF p, d;

-- name: ConfirmPayment :one
UPDATE payments
SET status = 'confirmed',
    confirmed_at = now()
WHERE id = $1
  AND status = 'pending_confirmation'
RETURNING id, debt_id, paid_by, received_by, amount_minor, status, note, confirmed_at, rejected_at, created_at, updated_at;

-- name: RejectPayment :one
UPDATE payments
SET status = 'rejected',
    rejected_at = now()
WHERE id = $1
  AND status = 'pending_confirmation'
RETURNING id, debt_id, paid_by, received_by, amount_minor, status, note, confirmed_at, rejected_at, created_at, updated_at;

-- name: ApplyConfirmedPaymentToDebt :one
UPDATE debts
SET remaining_amount_minor = remaining_amount_minor - $2,
    status = CASE
        WHEN remaining_amount_minor - $2 = 0 THEN 'settled'
        ELSE 'partially_settled'
    END,
    settled_at = CASE
        WHEN remaining_amount_minor - $2 = 0 THEN now()
        ELSE settled_at
    END
WHERE id = $1
  AND status IN ('accepted', 'partially_settled')
  AND remaining_amount_minor >= $2
RETURNING id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at;
