package quota

import "github.com/google/uuid"

type QuotaProvisioningEvent struct {
}

func PublishNotificationEvent(manager *QuotaManager, subscriberId uuid.UUID, notification string) {
	if manager == nil {
		return
	}
	manager.kafkaManager.PublishEvent("notification-event", subscriberId.String(), notification)
}
