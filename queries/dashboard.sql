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
