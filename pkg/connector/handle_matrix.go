package connector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/duo/matrix-pylon/pkg/ids"
	"github.com/duo/matrix-pylon/pkg/onebot"

	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

func (pc *PylonClient) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	if !pc.IsLoggedIn() {
		return nil, bridgev2.ErrNotLoggedIn
	}

	segments, err := pc.main.MsgConv.ToOnebot(ctx, pc.client, msg.Event, msg.Content, msg.Portal)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}

	if msg.ReplyTo != nil {
		if _, msgID, err := ids.ParseMessageID(msg.ReplyTo.ID); err != nil {
			return nil, err
		} else {
			segments = append([]onebot.ISegment{onebot.NewReply(msgID)}, segments...)
		}
	}

	peerType, peerID := ids.ParsePortalID(msg.Portal.ID)
	target, _ := strconv.ParseInt(peerID, 10, 64)

	var resp *onebot.SendMessageResponse

	switch peerType {
	case ids.PeerTypeUser:
		resp, err = pc.client.SendPrivateMessage(target, segments)
	case ids.PeerTypeGroup:
		resp, err = pc.client.SendGroupMessage(target, segments)
	default:
		return nil, fmt.Errorf("unsupported chat type %s", peerType)
	}

	if err != nil {
		return nil, bridgev2.WrapErrorInStatus(err).WithSendNotice(true)
	} else {
		_, peerID := ids.ParsePortalID(msg.Portal.ID)
		return &bridgev2.MatrixMessageResponse{
			DB: &database.Message{
				ID:        ids.MakeMessageID(peerID, fmt.Sprint(resp.MessageID)),
				SenderID:  networkid.UserID(pc.userLogin.ID),
				Timestamp: time.Now(),
			},
			StreamOrder: time.Now().Unix(),
		}, nil
	}
}

func (pc *PylonClient) HandleMatrixMessageRemove(ctx context.Context, msg *bridgev2.MatrixMessageRemove) error {

	_, messageID, err := ids.ParseMessageID(msg.TargetMessage.ID)
	if err != nil {
		return err
	}

	id, _ := strconv.ParseInt(messageID, 10, 32)

	return pc.client.DeleteMessage(int32(id))
}
