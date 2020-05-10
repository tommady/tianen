package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

var bot *linebot.Client

func main() {
	var err error
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_ACCESS_TOKEN"))
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/", callbackHandler)

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		fmt.Println(err)
	}
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		switch err {
		case linebot.ErrInvalidSignature:
			log.Println("400")
			w.WriteHeader(400)
		default:
			log.Println("500")
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.ImageMessage:
				log.Println("image")
				res, err := bot.GetMessageContent(message.ID).Do()
				if err != nil {
					log.Println(err)
					return
				}
				// Content       io.ReadCloser
				// ContentLength int64
				// ContentType   string
				fr := bufio.NewReader(res.Content)
				fo, err := os.Create("gglong.png")
				if err != nil {
					log.Println(err)
					return
				}
				defer fo.Close()

				wt := bufio.NewWriter(fo)
				buf := make([]byte, 1024)
				for {
					n, err := fr.Read(buf)
					if err != nil && err != io.EOF {
						log.Println(err)
						return
					}
					if n == 0 {
						break
					}
					if _, err := wt.Write(buf[:n]); err != nil {
						log.Println(err)
						return
					}
				}
				wt.Flush()
				log.Println(res.ContentLength, res.ContentType)
				log.Println(message.ID, message.OriginalContentURL, message.PreviewImageURL)
			case *linebot.TextMessage:
				log.Println("text")
			}
		} else if event.Type == linebot.EventTypeMemberJoined {
			log.Println("member joined")
		} else if event.Type == linebot.EventTypeJoin {
			log.Println("join")
		}
	}
}
