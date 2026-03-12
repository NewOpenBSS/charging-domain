package quota

import "github.com/google/uuid"

type QuotaProvisioningEvent struct {
}

func PublishNotificationEvent(manager *QuotaManager, subscriberId uuid.UUID, notification string) {

	manager.kafkaManager.PublishEvent("notification-event", subscriberId.String(), notification)
}
