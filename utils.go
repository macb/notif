package notif

import (
	"fmt"
	"strings"
)

func serviceStatus(node, checkID string) string {
	return fmt.Sprintf("notif/%s/%s", node, checkID)
}

func ignoredCheckID(checkID string) bool {
	ignore := []string{
		"_node_maintenance",
		"_service_maintenance",
	}

	for _, ignored := range ignore {
		if strings.HasPrefix(checkID, ignored) {
			return true
		}
	}

	return false
}
