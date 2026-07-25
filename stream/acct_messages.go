package stream

import "strings"

type acctFrame struct {
	Action    string   `json:"action"`
	Value     []string `json:"value,omitempty"`
	AuthToken string   `json:"auth-token"`
	RequestID int64    `json:"request-id"`
}

// bearer ensures exactly one "Bearer " prefix.
func bearer(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "Bearer ") {
		return token
	}
	return "Bearer " + token
}

func connectFrame(accounts []string, accessToken string, reqID int64) acctFrame {
	return acctFrame{Action: "connect", Value: accounts, AuthToken: bearer(accessToken), RequestID: reqID}
}

func heartbeatFrame(accessToken string, reqID int64) acctFrame {
	return acctFrame{Action: "heartbeat", AuthToken: bearer(accessToken), RequestID: reqID}
}
