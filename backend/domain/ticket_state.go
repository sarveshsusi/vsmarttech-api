package domain

import "rbac/models"

var ValidTransitions = map[models.TicketStatus][]models.TicketStatus{
	models.StatusOpen: {
		models.StatusAssigned,
		models.StatusClosed,
	},
	models.StatusAssigned: {
		models.StatusInProgress,
		models.StatusClosed,
	},
	models.StatusInProgress: {
		models.StatusClosed,
	},
	models.StatusClosed: {
		models.StatusAssigned, // admin reopen — keep engineer + PO
		models.StatusOpen,     // admin reopen when no engineer was assigned
	},
}

func CanTransition(from, to models.TicketStatus) bool {
	for _, s := range ValidTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
