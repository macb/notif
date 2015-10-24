package notif

import "fmt"

func serviceStatus(node, checkID string) string {
	return fmt.Sprintf("notif/%s/%s", node, checkID)
}
