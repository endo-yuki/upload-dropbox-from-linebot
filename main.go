package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"github.com/line/line-bot-sdk-go/linebot"
	"gopkg.in/ini.v1"
)

type Config struct {
	channelSecrt string
	channelToken string
	dropboxToken string
}

var conf Config

func init() {
	c, _ := ini.Load("config.ini")
	conf = Config{
		channelSecrt: c.Section("lineBotApi").Key("secret").String(),
		channelToken: c.Section("lineBotApi").Key("token").String(),
		dropboxToken: c.Section("DropboxApi").Key("token").String(),
	}
}

func main() {
	http.HandleFunc("/callback", uploadDropbox)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func uploadDropbox(w http.ResponseWriter, r *http.Request) {
	bot, err := linebot.New(
		conf.channelSecrt,
		conf.channelToken,
	)
	if err != nil {
		log.Fatal(err)
	}

	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.ImageMessage:
				content, err := bot.GetMessageContent(message.ID).Do()
				if err != nil {
					log.Println(err)
					return
				}
				filename := message.ID + ".png"
				file, err := os.Create(filename)
				if err != nil {
					log.Println(err)
					return
				}
				defer file.Close()

				if _, err = io.Copy(file, content.Content); err != nil {
					log.Println(err)
					return
				}

				config := dropbox.Config{
					Token: conf.dropboxToken,
				}
				cli := files.New(config)

				req := files.NewCommitInfo("/" + filename)
				f, _ := os.Open(filename)
				_, err = cli.Upload(req, f)
				if err != nil {
					log.Println(err)
					return
				}
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("アップロードが完了しました")).Do(); err != nil {
					log.Println(err)
					return
				}

				if err = file.Close(); err != nil {
					log.Println(err)
					return
				}
				if err = os.Remove(filename); err != nil {
					log.Println(err)
					return
				}

			default:
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("画像を送信してください")).Do(); err != nil {
					log.Println(err)
				}
			}
		}
	}
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}
