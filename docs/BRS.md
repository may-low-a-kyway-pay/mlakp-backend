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

### 3.0 Users and Groups
Users can:
- Register and log in
- Stay signed in through a secure session
- Log out of their current session
- View their own profile
- Create groups
- View groups they belong to
- Add members to groups they own

Group rules:
- A group has one creator.
- The group creator is the initial owner.
- A group member can create expenses for that group.
- Only a group owner can add members.
- A payer and all participants must be group members for group expenses.
- Historical expenses, debts, and payments remain visible to involved users even if membership rules are expanded later.

---

### 3.0.1 Authentication Sessions

Session rules:
- Registration and login create an authenticated session.
- Public authentication endpoints are rate-limited to reduce brute-force and abuse risk.
- Protected actions require an active authenticated session.
- Logout invalidates the current session.
- After logout, the same session must no longer allow access to protected data.
- Session expiry must eventually require the user to authenticate again.
- The system must not expose passwords, password hashes, access tokens, or refresh tokens in logs or API responses except when issuing new tokens to the authenticated client.

---

### 3.1 Expense Creation
Users can create an expense with:
- Title
- Total amount
- Payer
- Participants
- Split type (Equal / Manual)
- Optional: description, date, receipt

Rules:
- For MVP, expenses are created inside a group.
- Payer may be included in participants.
- If payer is included in participants, payer's own share does not create a self-debt.
- If payer is not included in participants, payer has no share.
- At least one participant other than the payer is required for the expense to create a debt.

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
- Debts are generated at expense creation time
- Debt amount equals the participant's share amount
- Creditor is the expense payer

---

### 3.4 Debt Acceptance

Debtors must:

- Accept → status becomes Accepted
- Reject → status becomes Rejected

If a debtor rejects:
- The rejected debt is excluded from financial totals while it remains rejected.
- A group owner can review the rejected debt.
- The group owner can resend it with the same amount or adjust the amount before resending.
- Resending moves the debt back to Pending so the debtor can accept or reject again.

---

### 3.5 Payment Recording

Debtor can:

- Mark full or partial payment

System creates a payment with:

```
Status: Pending Confirmation
```

---

### 3.6 Payment Review

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

Dashboard rules:
- Include only Accepted and Partially Settled debts.
- Exclude Pending, Rejected, and Settled debts.
- Use remaining debt amount, not original debt amount.

---

### 3.8 Expense Changes

For MVP:
- Expenses cannot be edited after creation.
- Expenses cannot be deleted after creation.
- If correction is needed, users must create a new expense or record the appropriate settlement outside the app.

Future versions may support cancellation or correction flows, but they must preserve audit history.

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
- If one debtor rejects and another accepts, only the accepted debt remains financially active until the rejected debt is reviewed and resent.
- Expenses cannot be edited or deleted in the MVP
- System must handle rounding differences to ensure total equals expense amount
- All amounts are assumed to be in a single currency
- A debt can have multiple payment records
- Total confirmed payments must not exceed debt amount
- Pending payments plus confirmed payments must not exceed remaining debt
- Only debtor can accept/reject debt
- Only debtor can mark payment
- Only creditor can confirm/reject payment
- Payer may be included in participants but does not create self-debt
- Payment and debt history must remain available for audit purposes
- Users must have an active authenticated session to access protected functionality
- Public authentication endpoints must return a rate-limit error when the client exceeds the allowed request window
- Logout must revoke the user's current session
- Users must not be able to view or mutate groups, expenses, debts, or payments they are not authorized to access

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

- User registration and login
- Session-backed logout
- Group creation and member management
- Expense creation **(equal + manual split)**
- Expense detail and group expense listing
- Debt tracking
- Debt acceptance/rejection
- Current-user debt listing
- Payment marking **(full/partial)**
- Payment review
- Basic dashboard **(You Owe / You Get)**

---
