package escalation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"czwlinux.cloud/go-friday-starter/features/auth"
	"czwlinux.cloud/go-friday-starter/features/project"
	"czwlinux.cloud/go-friday-starter/features/webhook"
	"czwlinux.cloud/go-friday-starter/global"
	"czwlinux.cloud/go-friday-starter/pkg/httpx/response"
)

// CreateEscalation creates a new escalation request.
func CreateEscalation(ctx context.Context, userID, projectID string, reason string) (*DTO, error) {
	if projectID == "" || reason == "" {
		return nil, fmt.Errorf("%w: project_id and reason are required", ErrInvalidInput)
	}

	// Default: 1 year from now (approver can shorten on approval)
	e := &Escalation{
		UserID:    userID,
		ProjectID: projectID,
		Reason:    reason,
		Status:    StatusPending,
		ExpiresAt: time.Now().Add(MaxDuration),
	}
	if err := Create(ctx, e); err != nil {
		return nil, err
	}
	webhook.SendWebhook(ctx, "escalation.created", map[string]interface{}{
		"escalation_id": e.ID,
		"user_id":       userID,
		"project_id":    projectID,
		"status":        e.Status,
	})
	return ToDTO(e), nil
}

// ApproveEscalation approves a pending escalation. Sets expiry to max 1 year from now.
// Project admin can self-approve.
func ApproveEscalation(ctx context.Context, escalationID, approverID string, duration string) (*DTO, error) {
	e, err := GetByID(ctx, escalationID)
	if err != nil {
		return nil, err
	}
	if e.Status != StatusPending {
		return nil, fmt.Errorf("escalation already %s", e.Status)
	}

	// Check approver permissions: must be project admin/approver, system_admin (*), or the applicant themselves (self-approval)
	userRole := project.GetUserProjectRole(ctx, e.ProjectID, approverID)
	isAdmin := userRole == project.MemberRoleAdmin || userRole == project.MemberRoleApprover
	isApplicant := approverID == e.UserID

	isSysAdmin := false
	codes, codeErr := auth.GetUserPermissionCodes(ctx, approverID)
	if codeErr == nil {
		isSysAdmin = auth.HasPermission(codes, "*")
	}

	if !isAdmin && !isSysAdmin && !isApplicant {
		return nil, fmt.Errorf("forbidden: only project admin/approver or system admin can approve escalations, or self-approval")
	}

	// Calculate expiry
	expiresAt := time.Now().Add(MaxDuration) // default 1 year
	if duration != "" {
		d, parseErr := time.ParseDuration(duration)
		if parseErr == nil && d > 0 && d <= MaxDuration {
			expiresAt = time.Now().Add(d)
		}
	}

	e.Status = StatusApproved
	e.ExpiresAt = expiresAt
	e.ApprovedBy = &approverID
	now := time.Now()
	e.ApprovedAt = &now

	if err := Save(ctx, e); err != nil {
		return nil, err
	}
	webhook.SendWebhook(ctx, "escalation.approved", map[string]interface{}{
		"escalation_id": e.ID,
		"user_id":       e.UserID,
		"project_id":    e.ProjectID,
		"approved_by":   approverID,
		"expires_at":    e.ExpiresAt,
	})
	return ToDTO(e), nil
}

// RejectEscalation rejects a pending escalation.
func RejectEscalation(ctx context.Context, escalationID, approverID string) (*DTO, error) {
	e, err := GetByID(ctx, escalationID)
	if err != nil {
		return nil, err
	}
	if e.Status != StatusPending {
		return nil, fmt.Errorf("escalation already %s", e.Status)
	}

	// Check approver permissions
	userRole := project.GetUserProjectRole(ctx, e.ProjectID, approverID)
	isAdmin := userRole == project.MemberRoleAdmin || userRole == project.MemberRoleApprover
	isApplicant := approverID == e.UserID

	isSysAdmin := false
	codes, codeErr := auth.GetUserPermissionCodes(ctx, approverID)
	if codeErr == nil {
		isSysAdmin = auth.HasPermission(codes, "*")
	}

	if !isAdmin && !isSysAdmin && !isApplicant {
		return nil, fmt.Errorf("forbidden: only project admin/approver or system admin can reject escalations, or self-rejection")
	}

	e.Status = StatusRejected
	e.RejectedBy = &approverID
	now := time.Now()
	e.RejectedAt = &now

	if err := Save(ctx, e); err != nil {
		return nil, err
	}
	webhook.SendWebhook(ctx, "escalation.rejected", map[string]interface{}{
		"escalation_id": e.ID,
		"user_id":       e.UserID,
		"project_id":    e.ProjectID,
		"rejected_by":   approverID,
	})
	return ToDTO(e), nil
}

// CheckActiveEscalation returns true if the user has an active (approved + not expired) escalation for the project.
func CheckActiveEscalation(ctx context.Context, userID, projectID string) (*ActiveResponse, error) {
	e, err := GetActiveByUserAndProject(ctx, userID, projectID)
	if errors.Is(err, ErrNoActiveEscalation) {
		return &ActiveResponse{Active: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &ActiveResponse{
		Active:     true,
		Escalation: ToDTO(e),
		ExpiresAt:  e.ExpiresAt,
	}, nil
}

// ListEscalations lists escalations based on query.
func ListEscalations(ctx context.Context, userID string, pq response.PageQuery, filters map[string]string) ([]*DTO, int64, error) {
	items, total, err := ListByScope(ctx, userID, pq, filters)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*DTO, 0, len(items))
	for i := range items {
		result = append(result, ToDTO(&items[i]))
	}
	return result, total, nil
}

// UpdateEscalation updates the reason for a pending escalation. Only the applicant can edit.
func UpdateEscalation(ctx context.Context, id, userID string, req UpdateRequest) (*DTO, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, ErrInvalidInput
	}

	e, err := GetByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if e.UserID != userID {
		return nil, ErrForbidden
	}
	if e.Status != StatusPending {
		return nil, fmt.Errorf("can only edit pending escalation")
	}

	e.Reason = reason
	if err := Save(ctx, e); err != nil {
		return nil, err
	}
	return ToDTO(e), nil
}

// DeleteEscalation soft-deletes a pending escalation. Only the applicant can delete.
func DeleteEscalation(ctx context.Context, id, userID string) error {
	if id == "" {
		return ErrInvalidInput
	}

	e, err := GetByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return err
	}
	if err != nil {
		return err
	}
	if e.UserID != userID {
		return ErrForbidden
	}
	if e.Status != StatusPending {
		return fmt.Errorf("can only delete pending escalation")
	}

	return DeleteByID(ctx, id)
}

// BatchDeleteEscalations batch-deletes pending escalations for the current user.
func BatchDeleteEscalations(ctx context.Context, userID string, ids []string) error {
	cleanIDs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		return ErrInvalidInput
	}

	// Only delete pending escalations owned by the user
	return global.DB.WithContext(ctx).
		Where("id IN ?", cleanIDs).
		Where("user_id = ?", userID).
		Where("status = ?", StatusPending).
		Delete(&Escalation{}).Error
}
