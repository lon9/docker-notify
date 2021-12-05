package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

const (
	// Start is identifier of start event
	Start = "start"
	// Die is identifier of die event
	Die = "die"
	// SlackURLEnv is key of SLACK_URL
	SlackURLEnv = "SLACK_URL"
	// DiscordURLEnv is key of DISCORD_URL
	DiscordURLEnv = "DISCORD_URL"
	// StartColor is color for started message
	StartColor = "#9ccc65"
	// DieColor is color for died message
	DieColor = "#c62828"
)

// Config is struct of config
type Config struct {
	SlackURL   string
	DiscordURL string
}

// NewConfig is constructor
func NewConfig() (*Config, error) {
	slackURL := os.Getenv(SlackURLEnv)
	discordURL := os.Getenv(DiscordURLEnv)
	if slackURL == "" && discordURL == "" {
		return nil, fmt.Errorf("%s and/or %s must be set", SlackURLEnv, DiscordURLEnv)
	}
	return &Config{
		SlackURL:   slackURL,
		DiscordURL: discordURL,
	}, nil
}

func main() {

	apiVersion := os.Getenv("API_VERSION")
	if apiVersion == "" {
		log.Fatal("API_VERSION must be set as your docker api version")
	}
	config, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	cli, err := client.NewClientWithOpts(client.WithVersion(apiVersion))
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	for {
		if err := start(cli, config); err != nil {
			log.Println(err)
		}
	}
}

func start(cli *client.Client, config *Config) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan, errChan := cli.Events(ctx, types.EventsOptions{})

L:
	for {
		select {
		case msg := <-msgChan:
			switch msg.Status {
			case Start:
				m, err := makeStartMessage(&msg)
				if err != nil {
					log.Println(err)
					continue
				}
				go m.Send(config)
			case Die:

				// Collect logs
				reader, err := cli.ContainerLogs(ctx, msg.ID, types.ContainerLogsOptions{
					Since:      "30s",
					ShowStdout: true,
					ShowStderr: true,
				})
				if err != nil {
					log.Println(err)
					continue
				}
				m, err := makeDieMessage(&msg, reader)
				if err != nil {
					log.Println(err)
					continue
				}
				go m.Send(config)
			}
		case err = <-errChan:
			break L
		}
	}
	return
}

func makeStartMessage(msg *events.Message) (m *Message, err error) {
	name, ok := msg.Actor.Attributes["name"]
	if !ok {
		return nil, errors.New("no name")
	}
	m = &Message{
		Attachments: []Attachment{
			{
				Title: fmt.Sprintf("Container started. name => %s image => %s", name, msg.From),
				Color: StartColor,
				TS:    msg.Time,
			},
		},
	}
	return
}

func makeDieMessage(msg *events.Message, logReder io.Reader) (m *Message, err error) {
	exitCode, ok := msg.Actor.Attributes["exitCode"]
	if !ok {
		return nil, errors.New("no exitCode")
	}
	name, ok := msg.Actor.Attributes["name"]
	if !ok {
		return nil, errors.New("no name")
	}
	m = &Message{
		Attachments: []Attachment{
			{
				Title: fmt.Sprintf("Container died. name => %s image => %s status code => %s", name, msg.From, exitCode),
				Color: DieColor,
				TS:    msg.Time,
			},
		},
	}
	b, err := ioutil.ReadAll(logReder)
	if err != nil {
		return nil, err
	}
	m.Attachments[0].Text = "```" + string(b) + "```"

	return
}

// Field is field of Attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Attachment is attachment of Message
type Attachment struct {
	Fallback   string  `json:"fallback"`
	Pretext    string  `json:"pretext"`
	Color      string  `json:"color"`
	Title      string  `json:"title"`
	TitleLink  string  `json:"title_link"`
	Text       string  `json:"text"`
	AuthorName string  `json:"author_name"`
	AuthorLink string  `json:"author_link"`
	AuthorIcon string  `json:"author_icon"`
	Footer     string  `json:"footer"`
	FooterIcon string  `json:"footer_icon"`
	TS         int64   `json:"ts"`
	Fields     []Field `json:"fields"`
}

// Message is struct of Slack's webhook
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// Send sends message to url
func (m *Message) Send(config *Config) {
	b, err := json.Marshal(m)
	if err != nil {
		log.Println(err)
		return
	}
	if config.SlackURL != "" {
		if err = m.post(config.SlackURL, b); err != nil {
			log.Println(err)
			return
		}
	}
	if config.DiscordURL != "" {
		if err = m.post(config.DiscordURL, b); err != nil {
			log.Println(err)
			return
		}
	}
}

func (m *Message) post(u string, body []byte) (err error) {
	resp, err := http.Post(u, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return
}
