package main

import (
	"bytes"
	"context"
	"encoding/json"
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
)

// Config is struct of config
type Config struct {
	SlackURL   string
	DiscordURL string
}

// NewConfig is constructor
func NewConfig() *Config {
	return &Config{
		SlackURL:   os.Getenv("SLACK_URL"),
		DiscordURL: os.Getenv("DISCORD_URL"),
	}
}

func main() {

	apiVersion := os.Getenv("API_VERSION")
	if apiVersion == "" {
		log.Fatal("API_VERSION must be set")
	}

	config := NewConfig()
	cli, err := client.NewClientWithOpts(client.WithVersion(apiVersion))
	if err != nil {
		panic(err)
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
				log.Println(msg)
				m := makeStartMessage(&msg)
				go m.Send(config)
			case Die:
				log.Println(msg)
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
				}
				go m.Send(config)
			}
		case err = <-errChan:
			break L
		}
	}
	return
}

func makeStartMessage(msg *events.Message) (m *Message) {
	m = &Message{
		Attachments: []Attachment{
			Attachment{
				Title: fmt.Sprintf("Container %s started", msg.From),
				Color: "#9ccc65",
				TS:    msg.Time,
			},
		},
	}
	return m
}

func makeDieMessage(msg *events.Message, logReder io.Reader) (m *Message, err error) {
	m = &Message{
		Attachments: []Attachment{
			Attachment{
				Title: fmt.Sprintf("Container %s died", msg.From),
				Color: "#c62828",
				TS:    msg.Time,
			},
		},
	}
	b, err := ioutil.ReadAll(logReder)
	if err != nil {
		return
	}
	m.Attachments[0].Text = string(b)

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
	Fallback   string  `json"fallback"`
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
	defer resp.Body.Close()
	return
}
