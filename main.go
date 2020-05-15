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

var (
	bot    *linebot.Client
	pool   = newWorkerPool(100, 10)
	mc     *minio.Client
	bucket = os.Getenv("MINIO_BUCKET")
)

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
	bot, err = linebot.New(os.Getenv("LINEBOT_CHANNEL_SECRET"), os.Getenv("LINEBOT_CHANNEL_ACCESS_TOKEN"))
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
			w.WriteHeader(400)
		default:
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		pool.PutJob(func() error {
			event := event

			if event.Source.Type != linebot.EventSourceTypeUser {
				return nil
			}

			if event.Source.UserID != "U00d6815afb6f57eb9ac959614d2520db" && event.Source.UserID != "U5b9f3b6e6636930d434372d70ce9c9b0" {
				return nil
			}

			var err error
			var res linebot.SendingMessage
			switch event.Type {
			case linebot.EventTypeMessage:
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					res, err = handleCommand(message.Text)
				case *linebot.ImageMessage:
					err = handleMessageContent(message.ID)
				case *linebot.VideoMessage:
					err = handleMessageContent(message.ID)
				}

			}
			if err != nil {
				return err
			}

			if _, err = bot.ReplyMessage(event.ReplyToken, res).Do(); err != nil {
				return err
			}
			return nil
		})
	}

}

const (
	todayObjectsCmd                = "[today objects]"
	thisMonthObjectsCmd            = "[this month objects]"
	defaultPresignedTimeoutMinutes = 30 * time.Minute
)

func handleCommand(msg string) (*linebot.TemplateMessage, error) {
	var prefix string
	switch msg {
	case todayObjectsCmd:
		prefix = time.Now().Format("2006-01-02")
	case thisMonthObjectsCmd:
		prefix = time.Now().Format("2006-01")
	default:
		return nil, nil
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	carousels := make([]*linebot.ImageCarouselColumn, 0)
	for obj := range mc.ListObjectsV2(bucket, prefix, false, doneCh) {
		url, err := mc.PresignedGetObject(bucket, obj.Key, defaultPresignedTimeoutMinutes, nil)
		if err != nil {
			return nil, err
		}

		carousels = append(carousels, linebot.NewImageCarouselColumn(url.String(), linebot.NewURIAction("", url.String())))
	}

	//(thumbnailImageURL, title, text string, actions ...TemplateAction)
	return linebot.NewTemplateMessage(
		"images",
		linebot.NewImageCarouselTemplate(carousels...),
	), nil
}

func handleMessageContent(msgID string) error {
	res, err := bot.GetMessageContent(msgID).Do()
	if err != nil {
		return err
	}

	_, err = mc.PutObject(bucket, time.Now().Format(time.RFC3339Nano), res.Content, res.ContentLength, minio.PutObjectOptions{
		ContentType: res.ContentType,
	})
	if err != nil {
		return err
	}

	return nil
}
