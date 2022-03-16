package slack

import (
	"context"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.uber.org/zap"
	"log"
	"os"
	"strings"
	"system-health-tool/internal/model"
	"system-health-tool/internal/report"
)

type Client struct {
	skClient       *slack.Client
	skSocketClient *socketmode.Client
	environments   model.Environments
}

func Run(ctx context.Context, environments model.Environments) {
	authToken := environments.SlackAuthToken
	appToken := environments.SlackAppToken

	skClient := slack.New(authToken, slack.OptionDebug(false), slack.OptionAppLevelToken(appToken))
	socketClient := socketmode.New(
		skClient,
		socketmode.OptionDebug(false),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)
	client := Client{skClient: skClient, skSocketClient: socketClient, environments: environments}

	go client.eventsListener(ctx)

	err := socketClient.Run()
	if err != nil {
		zap.S().Errorf("could not run socket skClient %v", err)
	}
}

func (client *Client) eventsListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			zap.S().Info("shutting down socket mode listener")
			return
		case event := <-client.skSocketClient.Events:
			switch event.Type {
			case socketmode.EventTypeEventsAPI:
				// The Event sent on the channel is not the same as the EventAPI events so we need to type cast it
				eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
				if !ok {
					zap.S().Errorf("could not type cast the event to the EventsAPIEvent: %v \n", event)
					continue
				}
				client.skSocketClient.Ack(*event.Request)
				client.handleEventMessage(ctx, eventsAPIEvent)
			}
		}
	}
}

func (client *Client) handleEventMessage(ctx context.Context, event slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			client.handleApplicationEvent(ctx, ev)
		}
	default:
		zap.S().Errorf("unsupported event type")
	}
}

func (client *Client) handleApplicationEvent(ctx context.Context, applicationEvent *slackevents.AppMentionEvent) {
	text := strings.ToLower(applicationEvent.Text)
	commands := strings.Fields(text)
	attachment := slack.Attachment{}
	attachment.Fields = []slack.AttachmentField{}

	if len(commands) == 3 || strings.Contains(strings.ToLower(commands[1]), "health") {
		targetSystem := commands[2]
		title, content := report.GetReportDetails(ctx, client.environments, targetSystem)
		attachment.Title = title
		attachment.Text = content
	}

	_, _, err := client.skClient.PostMessage(applicationEvent.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		zap.S().Errorf("failed to post message: %v", err)
	}
}
