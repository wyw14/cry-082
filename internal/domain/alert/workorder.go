package alert

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidWorkOrder = errors.New("invalid work order")

type WorkOrderStatus string

const (
	WorkOrderAssigned   WorkOrderStatus = "assigned"
	WorkOrderAccepted   WorkOrderStatus = "accepted"
	WorkOrderProcessing WorkOrderStatus = "processing"
	WorkOrderResolved   WorkOrderStatus = "resolved"
	WorkOrderVerified   WorkOrderStatus = "verified"
	WorkOrderCancelled  WorkOrderStatus = "cancelled"
)

type WorkOrder struct {
	ID          string
	AlertID     string
	AssigneeID  string
	Status      WorkOrderStatus
	Description string
	CreatedAt   time.Time
	DueAt       time.Time
	ResolvedAt  *time.Time
	Version     int64
}

func NewWorkOrder(id, alertID, assignee, description string, createdAt, dueAt time.Time) (WorkOrder, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(alertID) == "" || strings.TrimSpace(assignee) == "" || strings.TrimSpace(description) == "" || !createdAt.Before(dueAt) {
		return WorkOrder{}, ErrInvalidWorkOrder
	}
	return WorkOrder{ID: id, AlertID: alertID, AssigneeID: assignee, Status: WorkOrderAssigned, Description: description, CreatedAt: createdAt.UTC(), DueAt: dueAt.UTC(), Version: 1}, nil
}

func (w *WorkOrder) Transition(next WorkOrderStatus, at time.Time) error {
	allowed := map[WorkOrderStatus]map[WorkOrderStatus]bool{
		WorkOrderAssigned:   {WorkOrderAccepted: true, WorkOrderCancelled: true},
		WorkOrderAccepted:   {WorkOrderProcessing: true, WorkOrderCancelled: true},
		WorkOrderProcessing: {WorkOrderResolved: true},
		WorkOrderResolved:   {WorkOrderVerified: true, WorkOrderProcessing: true},
	}
	if !allowed[w.Status][next] {
		return ErrInvalidWorkOrder
	}
	w.Status = next
	if next == WorkOrderResolved {
		resolved := at.UTC()
		w.ResolvedAt = &resolved
	}
	w.Version++
	return nil
}
