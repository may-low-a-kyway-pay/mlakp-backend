## Database ER Diagram

The following diagram represents the database structure for the Shared Expense & Debt Tracking App.

It illustrates the relationships between users, groups, expenses, debts, and payments.  
This design supports key features such as expense splitting, debt tracking, partial payments, and settlement confirmation.

```mermaid
erDiagram
    USERS ||--o{ GROUPS : creates
    USERS ||--o{ GROUP_MEMBERS : joins
    GROUPS ||--o{ GROUP_MEMBERS : has

    USERS ||--o{ EXPENSES : creates
    USERS ||--o{ EXPENSES : pays
    GROUPS ||--o{ EXPENSES : contains

    EXPENSES ||--o{ EXPENSE_PARTICIPANTS : has
    USERS ||--o{ EXPENSE_PARTICIPANTS : participates

    EXPENSES ||--o{ DEBTS : generates
    USERS ||--o{ DEBTS : debtor
    USERS ||--o{ DEBTS : creditor

    DEBTS ||--o{ PAYMENTS : has
    USERS ||--o{ PAYMENTS : pays
    USERS ||--o{ PAYMENTS : receives

    USERS {
        uuid id PK "required"
        string name "required"
        string email UK "required"
        string password_hash "required"
        timestamp created_at "required"
        timestamp updated_at "required"
    }

    GROUPS {
        uuid id PK "required"
        string name "required"
        uuid created_by FK "required"
        timestamp created_at "required"
        timestamp updated_at "required"
    }

    GROUP_MEMBERS {
        uuid id PK "required"
        uuid group_id FK "required"
        uuid user_id FK "required"
        string role "required (default: member)"
        timestamp joined_at "required"
    }

    EXPENSES {
        uuid id PK "required"
        uuid group_id FK "optional"
        string title "required"
        text description "optional"
        decimal total_amount "required"
        string currency "required (default: THB)"
        uuid paid_by FK "required"
        string split_type "required (equal/manual)"
        text receipt_url "optional"
        date expense_date "optional"
        uuid created_by FK "required"
        timestamp created_at "required"
        timestamp updated_at "required"
    }

    EXPENSE_PARTICIPANTS {
        uuid id PK "required"
        uuid expense_id FK "required"
        uuid user_id FK "required"
        decimal share_amount "required"
        timestamp created_at "required"
    }

    DEBTS {
        uuid id PK "required"
        uuid expense_id FK "required"
        uuid debtor_id FK "required"
        uuid creditor_id FK "required"
        decimal original_amount "required"
        decimal remaining_amount "required"
        string status "required (pending/accepted/rejected/partially_settled/settled)"
        timestamp accepted_at "optional"
        timestamp rejected_at "optional"
        timestamp settled_at "optional"
        timestamp created_at "required"
        timestamp updated_at "required"
    }

    PAYMENTS {
        uuid id PK "required"
        uuid debt_id FK "required"
        uuid paid_by FK "required"
        uuid received_by FK "required"
        decimal amount "required"
        string status "required (pending_confirmation/confirmed/rejected)"
        text note "optional"
        timestamp confirmed_at "optional"
        timestamp rejected_at "optional"
        timestamp created_at "required"
        timestamp updated_at "required"
    }