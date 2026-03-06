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
	models.StatusClosed: {},
}

func CanTransition(from, to models.TicketStatus) bool {
	for _, s := range ValidTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
