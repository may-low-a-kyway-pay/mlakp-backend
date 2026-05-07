package groups

import "time"

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

type Group struct {
	ID        string
	Name      string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Member struct {
	ID       string
	GroupID  string
	UserID   string
	Role     string
	JoinedAt time.Time
	User     *MemberUser
}

type GroupDetails struct {
	Group   Group
	Members []Member
}

type MemberUser struct {
	ID       string
	Name     string
	Username string
	Email    string
}
