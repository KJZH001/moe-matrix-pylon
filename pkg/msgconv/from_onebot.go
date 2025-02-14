package msgconv

import (
	"context"
	"fmt"
	"html"
	"math"
	"path/filepath"
	"strings"

	"github.com/duo/matrix-pylon/pkg/ids"
	"github.com/duo/matrix-pylon/pkg/onebot"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func (mc *MessageConverter) OnebotToMatrix(
	ctx context.Context,
	client *onebot.Client,
	portal *bridgev2.Portal,
	intent bridgev2.MatrixAPI,
	msg *onebot.Message,
) *bridgev2.ConvertedMessage {
	ctx = context.WithValue(ctx, contextKeyClient, client)
	ctx = context.WithValue(ctx, contextKeyIntent, intent)
	ctx = context.WithValue(ctx, contextKeyPortal, portal)

	cm := &bridgev2.ConvertedMessage{}

	var part *bridgev2.ConvertedMessagePart

	mediaParts := make([]*bridgev2.ConvertedMessagePart, 0)
	mentions := make([]string, 0)

	var contentBuilder strings.Builder

	for _, s := range msg.Message.([]onebot.ISegment) {
		switch v := s.(type) {
		case *onebot.TextSegment:
			fmt.Fprint(&contentBuilder, v.Content())
		case *onebot.FaceSegment:
			fmt.Fprintf(&contentBuilder, "/[Face%s]", v.ID())
		case *onebot.AtSegment:
			target := v.Target()
			if target == "all" {
				target = "room" // Matrix's mention all
			}
			fmt.Fprintf(&contentBuilder, "@%s", target)
			mentions = append(mentions, target)
		case *onebot.ImageSegment:
			mediaParts = append(mediaParts, mc.convertMediaMessage(ctx, v))
			fmt.Fprint(&contentBuilder, "[Image]")
		case *onebot.MarketFaceSegment:
			mediaParts = append(mediaParts, mc.convertMediaMessage(ctx, v))
			fmt.Fprint(&contentBuilder, "[Image]")
		case *onebot.RecordSegment:
			mediaParts = append(mediaParts, mc.convertMediaMessage(ctx, v))
			fmt.Fprint(&contentBuilder, "[Voice]")
		case *onebot.VideoSegment:
			mediaParts = append(mediaParts, mc.convertMediaMessage(ctx, v))
			fmt.Fprint(&contentBuilder, "[Video]")
		case *onebot.FileSegment:
			mediaParts = append(mediaParts, mc.convertMediaMessage(ctx, v))
			fmt.Fprint(&contentBuilder, "[File]")
		case *onebot.ReplySegment:
			cm.ReplyTo = &networkid.MessageOptionalPartID{
				MessageID: ids.MakeMessageID(ids.GetPeerID(msg), v.ID()),
			}
		case *onebot.ForwardSegment:
			fmt.Fprint(&contentBuilder, "[Chat History]")
		case *onebot.ShareSegment:
			part = mc.convertShareMessage(v.Title(), v.Content(), v.URL())
		case *onebot.JSONSegment:
			part = mc.convertJSONMessage(ctx, v)
		default:
			fmt.Fprintf(&contentBuilder, "[%s]", v.SegmentType())
		}
	}

	if part == nil {
		if len(mediaParts) > 1 {
			var imagesMarkdown strings.Builder
			for _, part := range mediaParts {
				fmt.Fprintf(&imagesMarkdown, "![%s](%s)\n", part.Content.FileName, part.Content.URL)
			}

			rendered := format.RenderMarkdown(imagesMarkdown.String(), true, false)
			content := contentBuilder.String()
			part = &bridgev2.ConvertedMessagePart{
				Type: event.EventMessage,
				Content: &event.MessageEventContent{
					MsgType:       event.MsgText,
					Format:        event.FormatHTML,
					Body:          content,
					FormattedBody: fmt.Sprintf("%s\n%s", rendered.FormattedBody, content),
				},
			}
		} else if len(mediaParts) == 1 {
			part = mediaParts[0]
		} else {
			part = &bridgev2.ConvertedMessagePart{
				Type: event.EventMessage,
				Content: &event.MessageEventContent{
					MsgType: event.MsgText,
					Body:    contentBuilder.String(),
				},
			}
		}
	}

	// Mentions
	part.Content.Mentions = &event.Mentions{}
	mc.addMentions(ctx, mentions, part.Content)

	cm.Parts = []*bridgev2.ConvertedMessagePart{part}

	return cm
}

func (mc *MessageConverter) convertMediaMessage(ctx context.Context, seg onebot.ISegment) *bridgev2.ConvertedMessagePart {
	if part, err := mc.reploadAttachment(ctx, seg); err != nil {
		return mc.makeMediaFailure(ctx, err)
	} else {
		return part
	}
}

func (mc *MessageConverter) convertJSONMessage(_ context.Context, seg *onebot.JSONSegment) *bridgev2.ConvertedMessagePart {
	content := seg.Content()

	view := gjson.Get(content, "view").String()
	if view == "LocationShare" {
		name := gjson.Get(content, "meta.*.name").String()
		address := gjson.Get(content, "meta.*.address").String()
		latitude := gjson.Get(content, "meta.*.lat").Float()
		longitude := gjson.Get(content, "meta.*.lng").Float()

		return mc.convertLocationMessage(name, address, latitude, longitude)
	} else {
		if url := gjson.Get(content, "meta.*.qqdocurl").String(); len(url) > 0 {
			desc := gjson.Get(content, "meta.*.desc").String()
			prompt := gjson.Get(content, "prompt").String()
			return mc.convertShareMessage(prompt, desc, url)
		} else if url := gjson.Get(content, "meta.*.jumpUrl").String(); len(url) > 0 {
			desc := gjson.Get(content, "meta.*.desc").String()
			prompt := gjson.Get(content, "prompt").String()
			return mc.convertShareMessage(prompt, desc, url)
		}
	}

	return &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    content,
		},
	}
}

func (mc *MessageConverter) convertLocationMessage(name, address string, latitude, longitude float64) *bridgev2.ConvertedMessagePart {
	url := fmt.Sprintf("https://maps.google.com/?q=%.5f,%.5f", latitude, longitude)
	if len(name) == 0 {
		latChar := 'N'
		if latitude < 0 {
			latChar = 'S'
		}
		longChar := 'E'
		if longitude < 0 {
			longChar = 'W'
		}
		name = fmt.Sprintf("%.4f° %c %.4f° %c", math.Abs(latitude), latChar, math.Abs(longitude), longChar)
	}

	content := &event.MessageEventContent{
		MsgType:       event.MsgLocation,
		Body:          fmt.Sprintf("Location: %s\n%s\n%s", name, address, url),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("Location: <a href='%s'>%s</a><br>%s", url, name, address),
		GeoURI:        fmt.Sprintf("geo:%.5f,%.5f", latitude, longitude),
	}

	return &bridgev2.ConvertedMessagePart{
		Type:    event.EventMessage,
		Content: content,
	}
}

func (mc *MessageConverter) convertShareMessage(title, desc, url string) *bridgev2.ConvertedMessagePart {
	body := fmt.Sprintf("%s\n\n%s\n\n%s", title, desc, url)
	rendered := format.RenderMarkdown(
		fmt.Sprintf("**%s**\n%s\n\n[%s](%s)", title, desc, url, url),
		true,
		false,
	)

	return &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			Body:          body,
			MsgType:       event.MsgText,
			Format:        event.FormatHTML,
			FormattedBody: rendered.FormattedBody,
		},
	}
}

func (mc *MessageConverter) reploadAttachment(ctx context.Context, seg onebot.ISegment) (*bridgev2.ConvertedMessagePart, error) {
	content := &event.MessageEventContent{
		Info: &event.FileInfo{},
	}

	fileName, data, err := getClient(ctx).DownloadMedia(seg)
	if err != nil {
		return nil, fmt.Errorf("failed to download attachment: %w", err)
	}

	mime := mimetype.Detect(data)
	if filepath.Ext(fileName) == "" {
		fileName = fileName + mime.Extension()
	}
	content.Info.Size = len(data)
	content.FileName = fileName

	content.URL, content.File, err = getIntent(ctx).UploadMedia(ctx, getPortal(ctx).MXID, data, fileName, mime.String())
	if err != nil {
		return nil, err
	}

	switch seg.(type) {
	case *onebot.ImageSegment:
		content.MsgType = event.MsgImage
	case *onebot.MarketFaceSegment:
		content.MsgType = event.MsgImage
	case *onebot.VideoSegment:
		content.MsgType = event.MsgVideo
	case *onebot.FileSegment:
		content.MsgType = event.MsgFile
	case *onebot.RecordSegment:
		content.MsgType = event.MsgAudio
		content.MSC3245Voice = &event.MSC3245Voice{}
	}

	//content.Body = fileName
	content.Info.MimeType = mime.String()

	return &bridgev2.ConvertedMessagePart{
		Type:    event.EventMessage,
		Content: content,
	}, nil
}

func (mc *MessageConverter) makeMediaFailure(ctx context.Context, err error) *bridgev2.ConvertedMessagePart {
	zerolog.Ctx(ctx).Err(err).Msg("Failed to reupload Onebot attachment")
	return &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgNotice,
			Body:    fmt.Sprintf("Failed to upload Onebot attachment"),
		},
	}
}

func (mc *MessageConverter) addMentions(ctx context.Context, mentions []string, into *event.MessageEventContent) {
	if len(mentions) == 0 {
		return
	}

	into.EnsureHasHTML()

	for _, id := range mentions {
		if id == "room" {
			into.Mentions.Room = true
			continue
		}

		// TODO: get group nickname
		mxid, displayname, err := mc.getBasicUserInfo(ctx, ids.MakeUserID(id))
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Str("id", id).Msg("Failed to get user info")
			continue
		}
		into.Mentions.UserIDs = append(into.Mentions.UserIDs, mxid)
		mentionText := "@" + id
		into.Body = strings.ReplaceAll(into.Body, mentionText, displayname)
		into.FormattedBody = strings.ReplaceAll(into.FormattedBody, mentionText, fmt.Sprintf(`<a href="%s">%s</a>`, mxid.URI().MatrixToURL(), html.EscapeString(displayname)))
	}
}

func (mc *MessageConverter) getBasicUserInfo(ctx context.Context, user networkid.UserID) (id.UserID, string, error) {
	ghost, err := mc.Bridge.GetGhostByID(ctx, user)
	if err != nil {
		return "", "", fmt.Errorf("failed to get ghost by ID: %w", err)
	}
	login := mc.Bridge.GetCachedUserLoginByID(networkid.UserLoginID(user))
	if login != nil {
		return login.UserMXID, ghost.Name, nil
	}
	return ghost.Intent.GetMXID(), ghost.Name, nil
}
