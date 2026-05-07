package handlers

import (
	"errors"
	"net/http"

	"mlakp-backend/internal/groups"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
)

type GroupHandler struct {
	groups *groups.Service
}

type groupResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type groupMemberResponse struct {
	ID       string                   `json:"id"`
	GroupID  string                   `json:"group_id"`
	UserID   string                   `json:"user_id"`
	Role     string                   `json:"role"`
	JoinedAt string                   `json:"joined_at"`
	User     *groupMemberUserResponse `json:"user,omitempty"`
}

type groupMemberUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewGroupHandler(groups *groups.Service) *GroupHandler {
	return &GroupHandler{groups: groups}
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	group, err := h.groups.Create(r.Context(), request.Name, userID)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	response.Success(w, http.StatusCreated, map[string]groupResponse{
		"group": toGroupResponse(group),
	})
}

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	groupList, err := h.groups.ListForUser(r.Context(), userID)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	groupsResponse := make([]groupResponse, 0, len(groupList))
	for _, group := range groupList {
		groupsResponse = append(groupsResponse, toGroupResponse(group))
	}

	response.Success(w, http.StatusOK, map[string][]groupResponse{
		"groups": groupsResponse,
	})
}

func (h *GroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	details, err := h.groups.GetForUser(r.Context(), r.PathValue("groupID"), userID)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	members := make([]groupMemberResponse, 0, len(details.Members))
	for _, member := range details.Members {
		members = append(members, toGroupMemberResponse(member))
	}

	response.Success(w, http.StatusOK, map[string]any{
		"group":   toGroupResponse(details.Group),
		"members": members,
	})
}

func (h *GroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request struct {
		UserID string `json:"user_id"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	member, err := h.groups.AddMember(r.Context(), r.PathValue("groupID"), userID, request.UserID)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	response.Success(w, http.StatusCreated, map[string]groupMemberResponse{
		"member": toGroupMemberResponse(member),
	})
}

func writeGroupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, groups.ErrInvalidName):
		response.Error(w, http.StatusBadRequest, "invalid_group_name", "Group name must be between 1 and 120 characters")
	case errors.Is(err, groups.ErrInvalidGroupID):
		response.Error(w, http.StatusBadRequest, "invalid_group_id", "Group ID is invalid")
	case errors.Is(err, groups.ErrInvalidUserID):
		response.Error(w, http.StatusBadRequest, "invalid_user_id", "User ID is invalid")
	case errors.Is(err, groups.ErrNotFound):
		response.Error(w, http.StatusNotFound, "group_not_found", "Group was not found")
	case errors.Is(err, groups.ErrForbidden):
		response.Error(w, http.StatusForbidden, "group_forbidden", "You are not allowed to modify this group")
	case errors.Is(err, groups.ErrMemberNotFound):
		response.Error(w, http.StatusNotFound, "group_member_user_not_found", "Group member user was not found")
	case errors.Is(err, groups.ErrMemberConflict):
		response.Error(w, http.StatusConflict, "group_member_already_exists", "User is already a group member")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toGroupResponse(group groups.Group) groupResponse {
	return groupResponse{
		ID:        group.ID,
		Name:      group.Name,
		CreatedBy: group.CreatedBy,
		CreatedAt: group.CreatedAt.Format(timeFormatRFC3339),
		UpdatedAt: group.UpdatedAt.Format(timeFormatRFC3339),
	}
}

func toGroupMemberResponse(member groups.Member) groupMemberResponse {
	response := groupMemberResponse{
		ID:       member.ID,
		GroupID:  member.GroupID,
		UserID:   member.UserID,
		Role:     member.Role,
		JoinedAt: member.JoinedAt.Format(timeFormatRFC3339),
	}

	if member.User != nil {
		response.User = &groupMemberUserResponse{
			ID:    member.User.ID,
			Name:  member.User.Name,
			Email: member.User.Email,
		}
	}

	return response
}
