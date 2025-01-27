package ids

import (
	"fmt"
	"strings"

	"github.com/duo/matrix-pylon/pkg/onebot"

	"maunium.net/go/mautrix/bridgev2/networkid"
)

type PeerType string

const (
	PeerTypeUser  PeerType = "user"
	PeerTypeGroup PeerType = "group"
)

func ParsePortalID(portalID networkid.PortalID) (PeerType, string) {
	parts := strings.SplitN(string(portalID), "\u0001", 2)
	if len(parts) == 1 {
		return PeerTypeUser, parts[0]
	}
	return PeerType(parts[0]), parts[1]
}

func MakeUserID(id string) networkid.UserID {
	return networkid.UserID(id)
}

func MakeUserLoginID(id string) networkid.UserLoginID {
	return networkid.UserLoginID(id)
}

func MakeMessageID(peerID string, msgID string) networkid.MessageID {
	return networkid.MessageID(fmt.Sprintf("%s:%s", peerID, msgID))
}

func MakeFakeMessageID(peerID string, data string) networkid.MessageID {
	return networkid.MessageID(fmt.Sprintf("fake:%s:%s", peerID, data))
}

func ParseMessageID(messageID networkid.MessageID) (string, string, error) {
	parts := strings.SplitN(string(messageID), ":", 2)
	if len(parts) == 2 {
		if parts[0] == "fake" {
			return "", "", fmt.Errorf("fake message ID")
		}
		return parts[0], parts[1], nil
	} else {
		return "", "", fmt.Errorf("invalid message ID")
	}
}
func GetPeerID(message *onebot.Message) string {
	var peerID int64

	if message.EventType() == onebot.MessagePrivate {
		peerID = message.Sender.UserID
		if message.PostType == "message_sent" { // Sent by self
			peerID = message.TargetID
		}
	} else {
		peerID = message.GroupID
	}

	return fmt.Sprint(peerID)
}
