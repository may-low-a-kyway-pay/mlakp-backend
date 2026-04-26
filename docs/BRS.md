# Business Requirement Specification (BRS)
## Shared Expense & Debt Tracking App

## 1. Overview

This application helps users track shared expenses and unsettled debts within a group (e.g., friends living together).

Users can:
- Record expenses
- Split costs (equal or manual)
- Track who owes whom
- Confirm debts and settlements

The system does **not handle real money transfers**. It only records and tracks obligations.

---

## 2. Core Concepts (Terminology)

- **Expense**: A payment made by one user for a group
- **Payer**: User who paid the expense
- **Participant**: Users sharing the expense
- **Share**: Individual portion of the expense
- **Debt**: Amount a user owes another (Debtor → Creditor)
- **Debtor**: User who owes money
- **Creditor**: User who should receive money
- **Settlement**: Confirmation that a debt is paid (outside the app)

---

## 3. Functional Requirements

### 3.1 Expense Creation
Users can create an expense with:
- Title
- Total amount
- Payer
- Participants
- Split type (Equal / Manual)
- Optional: description, date, receipt

---

### 3.2 Expense Split

#### Equal Split
- Amount divided equally among participants

#### Manual Split
- User defines share per participant
- Validation required:
```text
Sum of shares = Total amount
```

---

### 3.3 Debt Generation

- System generates debts for each participant (excluding payer’s share)
- Initial status: **Pending**

---

### 3.4 Debt Acceptance

Debtors must:

- Accept → status becomes Accepted
- Reject → status becomes Rejected

---

### 3.5 Payment Recording

Debtor can:

- Mark full or partial payment

System creates a payment with:

```
Status: Pending Confirmation
```

---

### 3.6 Payment Confirmation

Creditor can:

- Confirm → update debt (partial or full settlement)
- Reject → no change to debt

---

### 3.7 Dashboard

Users can view:

#### You Owe:
- Total amount the user owes others
#### You Get:
- Total amount others owe the user

---

## 4. Status Definitions

#### Debt Status

- Pending
- Accepted
- Rejected
- Partially Settled
- Settled

#### Payment Status

- Pending Confirmation
- Confirmed
- Rejected

---

## 5. Business Rules

- Payer’s own share does not create a debt
- Total split must equal total expense
- Debt must be accepted to become valid
- Payment must be confirmed to be valid
- Partial payments reduce remaining debt
- Remaining = 0 → Settled, Remaining > 0 → Partially Settled
- Rejected debts are excluded
- Rejected payments do not affect debt amount
- Payment record remains for history
- System does not process real money
- Pending debts (not accepted) must not be included in dashboard totals
- If all debtors reject, the expense has no financial effect
- Expenses cannot be edited or deleted once any debt is accepted
- System must handle rounding differences to ensure total equals expense amount
- All amounts are assumed to be in a single currency
- A debt can have multiple payment records
- Total confirmed payments must not exceed debt amount
- Only debtor can accept/reject debt
- Only debtor can mark payment
- Only creditor can confirm/reject payment
- Payer may be included in participants but does not create self-debt

--- 

## 6. User Flow

#### Expense Flow

```
Create Expense → Split → Generate Debts → Debtors Accept/Reject
```

#### Settlement Flow

```
Debtor Pays → Mark as Paid → Creditor Confirms → Debt Updated
```

---

## 7. MVP Scope

- Expense creation **(equal + manual split)**
- Debt tracking
- Debt acceptance/rejection
- Payment marking **(full/partial)**
- Payment confirmation
- Basic dashboard **(You Owe / You Get)**

---
