-- name: GetDashboardTotalsForUser :one
SELECT
    COALESCE(SUM(remaining_amount_minor) FILTER (
        WHERE debtor_id = $1
          AND status IN ('accepted', 'partially_settled')
    ), 0)::bigint AS you_owe_minor,
    COALESCE(COUNT(*) FILTER (
        WHERE debtor_id = $1
          AND status IN ('accepted', 'partially_settled')
    ), 0)::bigint AS you_owe_count,
    COALESCE(SUM(remaining_amount_minor) FILTER (
        WHERE creditor_id = $1
          AND status IN ('accepted', 'partially_settled')
    ), 0)::bigint AS you_get_minor,
    COALESCE(COUNT(*) FILTER (
        WHERE creditor_id = $1
          AND status IN ('accepted', 'partially_settled')
    ), 0)::bigint AS you_get_count
FROM debts
WHERE debtor_id = $1
   OR creditor_id = $1;

-- name: ListDashboardUnsettledBalances :many
SELECT
    d.id,
    d.expense_id,
    e.title AS expense_title,
    d.debtor_id,
    debtor.name AS debtor_name,
    d.creditor_id,
    creditor.name AS creditor_name,
    d.remaining_amount_minor,
    d.status,
    d.updated_at
FROM debts d
JOIN expenses e ON e.id = d.expense_id
JOIN users debtor ON debtor.id = d.debtor_id
JOIN users creditor ON creditor.id = d.creditor_id
WHERE (d.debtor_id = $1 OR d.creditor_id = $1)
  AND d.status IN ('accepted', 'partially_settled')
ORDER BY d.updated_at DESC, d.created_at DESC, d.id DESC
LIMIT 5;
