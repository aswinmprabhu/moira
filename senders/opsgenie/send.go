package opsgenie

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

const (
	titleLimit = 130
	msgLimit   = 15000
)

// SendEvents sends the events as an alert to opsgenie
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	createAlertRequest := sender.makeCreateAlertRequest(events, contact, trigger, plot, throttled)
	_, err := sender.client.Create(nil, createAlertRequest)
	if err != nil {
		return fmt.Errorf("error while creating alert: %s", err)
	}
	if err != nil {
		return fmt.Errorf("failed to send %s event message to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
	}
	return nil
}

func (sender *Sender) makeCreateAlertRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) *alert.CreateAlertRequest {
	createAlertRequest := &alert.CreateAlertRequest{
		Message:     sender.buildTitle(events, trigger),
		Description: sender.buildMessage(events, throttled, trigger),
		// Responders: []alert.Responder{
		// 	{Type: alert.EscalationResponder, Name: "TeamA_escalation"},
		// 	{Type: alert.ScheduleResponder, Name: "TeamB_schedule"},
		// },
		Tags:     trigger.Tags,
		Entity:   "entity2",
		Source:   "Moira",
		Priority: sender.getMessagePriority(events),
	}
	return createAlertRequest
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, throttled bool, trigger moira.TriggerData) string {
	var message strings.Builder

	desc := trigger.Desc
	htmlDesc := string(blackfriday.Run([]byte(desc)))
	htmlDescLen := len([]rune(htmlDesc))
	charsForHTMLTags := htmlDescLen - len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(msgLimit, htmlDescLen, eventsStringLen)

	if htmlDescLen != descNewLen {
		desc = desc[:descNewLen-charsForHTMLTags] + "...\n"
		htmlDesc = string(blackfriday.Run([]byte(desc)))
	}
	if eventsNewLen != eventsStringLen {
		eventsString = sender.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(htmlDesc)
	message.WriteString(eventsString)
	return message.String()
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *Sender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, fix your system or tune this trigger to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s\n", moira.UseString(event.Message))
		} else {
			line += "\n"
		}

		tailStringLen := len([]rune(fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}

	if eventsLenLimitReached {
		eventsString += fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("%s %s %s (%d)", events.GetSubjectState(), trigger.Name, trigger.GetTags(), len(events))
	tags := 1
	for len([]rune(title)) > titleLimit {
		var tagBuffer bytes.Buffer
		for i := 0; i < len(trigger.Tags)-tags; i++ {
			tagBuffer.WriteString(fmt.Sprintf("[%s]", trigger.Tags[i]))
		}
		title = fmt.Sprintf("%s %s %s.... (%d)", events.GetSubjectState(), trigger.Name, tagBuffer.String(), len(events))
		tags++
	}
	return title
}

func (sender *Sender) getMessagePriority(events moira.NotificationEvents) alert.Priority {
	priority := alert.P5
	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			priority = alert.P1
		}
		if priority != alert.P1 && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			priority = alert.P3
		}
	}
	return priority
}