package connector

import (
	"fmt"

	"github.com/duo/matrix-pylon/pkg/ids"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

func (pc *PylonClient) selfEventSender() bridgev2.EventSender {
	return bridgev2.EventSender{
		IsFromMe:    true,
		Sender:      networkid.UserID(pc.userLogin.ID),
		SenderLogin: pc.userLogin.ID,
	}
}

func (pc *PylonClient) makeEventSender(id string) bridgev2.EventSender {
	return bridgev2.EventSender{
		IsFromMe:    ids.MakeUserLoginID(id) == pc.userLogin.ID,
		Sender:      ids.MakeUserID(id),
		SenderLogin: ids.MakeUserLoginID(id),
	}
}

func (pc *PylonClient) makePortalKey(peerType ids.PeerType, peerID string) networkid.PortalKey {
	key := networkid.PortalKey{}
	if peerType == ids.PeerTypeGroup {
		key.ID = networkid.PortalID(fmt.Sprintf("%s\u0001%s", peerType, peerID))
	} else {
		key.ID = networkid.PortalID(peerID)
		// For non-group chats, add receiver
		key.Receiver = pc.userLogin.ID
	}
	return key
}

func (pc *PylonClient) makeDMPortalKey(identifier string) networkid.PortalKey {
	return networkid.PortalKey{
		ID:       networkid.PortalID(identifier),
		Receiver: pc.userLogin.ID,
	}
}
