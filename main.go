package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/minio/minio-go/v6"
)

var bot *linebot.Client
var pool = newWorkerPool(100, 10)
var mc *minio.Client

type jobFunc func() error

type workerPool struct {
	wg   *sync.WaitGroup
	pool chan jobFunc
}

func newWorkerPool(poolSize, workerNum int) *workerPool {
	wp := &workerPool{
		wg:   new(sync.WaitGroup),
		pool: make(chan jobFunc, poolSize),
	}

	for i := 0; i < workerNum; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *workerPool) worker() {
	for job := range wp.pool {
		if err := job(); err != nil {
			log.Println(err)
		}
	}
}

func (wp *workerPool) PutJob(job jobFunc) {
	wp.pool <- job
}

func (wp *workerPool) Close() {
	close(wp.pool)
	wp.wg.Wait()
}

func main() {
	var err error
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_ACCESS_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	mc, err = minio.New(os.Getenv("MINIO_ENDPOINT"), os.Getenv("MINIO_ACCESS_KEY_ID"), os.Getenv("MINIO_SECRET_ACCESS_KEY"), false)
	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	http.HandleFunc("/", callbackHandler)

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
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
		pool.PutJob(func() error {
			event := event
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.ImageMessage:
					log.Println("image")
					res, err := bot.GetMessageContent(message.ID).Do()
					if err != nil {
						return err
					}

					_, err = mc.PutObject("photo", time.Now().Format(time.RFC3339), res.Content, res.ContentLength, minio.PutObjectOptions{
						ContentType: res.ContentType,
					})
					if err != nil {
						return err
					}

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
			return nil
		})
	}

}
