package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/notifications"

	"github.com/coder/websocket"
)

type NotificationHandler struct {
	notifications *notifications.Service
	hub           *notifications.Hub
	origins       []string
}

type notificationResponse struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Metadata   json.RawMessage `json:"metadata"`
	ReadAt     *string         `json:"read_at"`
	CreatedAt  string          `json:"created_at"`
}

type realtimeClient struct {
	conn *websocket.Conn
}

func NewNotificationHandler(notifications *notifications.Service, hub *notifications.Hub, origins []string) *NotificationHandler {
	return &NotificationHandler{notifications: notifications, hub: hub, origins: origins}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	notificationList, unreadCount, err := h.notifications.List(r.Context(), notifications.ListInput{
		UserID: userID,
		Limit:  parseNotificationLimit(r.URL.Query().Get("limit")),
	})
	if err != nil {
		writeNotificationError(w, err)
		return
	}

	notificationsResponse := make([]notificationResponse, 0, len(notificationList))
	for _, notification := range notificationList {
		notificationsResponse = append(notificationsResponse, toNotificationResponse(notification))
	}

	response.Success(w, http.StatusOK, map[string]any{
		"notifications": notificationsResponse,
		"unread_count":  unreadCount,
	})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	notification, unreadCount, err := h.notifications.MarkRead(r.Context(), notifications.MarkReadInput{
		ID:     r.PathValue("notificationID"),
		UserID: userID,
	})
	if err != nil {
		writeNotificationError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]any{
		"notification": toNotificationResponse(notification),
		"unread_count": unreadCount,
	})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	unreadCount, err := h.notifications.MarkAllRead(r.Context(), userID)
	if err != nil {
		writeNotificationError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]int64{
		"unread_count": unreadCount,
	})
}

func (h *NotificationHandler) Realtime(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.origins,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "closing realtime connection")

	client := &realtimeClient{conn: conn}
	unsubscribe := h.hub.Subscribe(userID, client)
	defer unsubscribe()

	// The server only pushes notification events. Reads keep the connection open
	// until the client disconnects or the request context is cancelled.
	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			return
		}
	}
}

func (c *realtimeClient) Send(payload []byte) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.conn.Write(ctx, websocket.MessageText, payload) == nil
}

func parseNotificationLimit(value string) int32 {
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 50
	}
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return int32(limit)
}

func toNotificationResponse(notification notifications.Notification) notificationResponse {
	metadata := json.RawMessage(notification.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage("{}")
	}

	return notificationResponse{
		ID:         notification.ID,
		UserID:     notification.UserID,
		Type:       notification.Type,
		Title:      notification.Title,
		Body:       notification.Body,
		EntityType: notification.EntityType,
		EntityID:   notification.EntityID,
		Metadata:   metadata,
		ReadAt:     optionalTimeString(notification.ReadAt),
		CreatedAt:  notification.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func optionalTimeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func writeNotificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notifications.ErrInvalidNotificationID):
		response.Error(w, http.StatusBadRequest, "invalid_notification_id", "Notification ID is invalid")
	case errors.Is(err, notifications.ErrInvalidUserID):
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
	case errors.Is(err, notifications.ErrNotFound):
		response.Error(w, http.StatusNotFound, "notification_not_found", "Notification was not found")
	default:
		response.Error(w, http.StatusInternalServerError, "notification_error", "Unable to process notification request")
	}
}
