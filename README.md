# docker-notify

Notifying docker started/died event to Slack and/or Discord


1. Edit docker-notify.env for your environment. Make sure put `/slack` on end of `DISCORD_URL`, because it send discord webhook with message structure of Slack.

1. Start docker-compose

```
docker-compose up -d
```
