package preview

import (
	context "context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func (s *Server) do(id string) {
	s.work <- id
}

type resp struct {
	id   string
	data []byte
	err  error
}

func (s *Server) listen() {
	//only one unit of work at a time
	var queue []string
	done := make(chan resp)
	count := 0

	for {
		select {
		case w := <-s.work:
			queue = append(queue, w)
			s.logger.Info("got work", "id", w)
		case res := <-done:
			count--
			var data string
			msg := "done"
			if res.err != nil {
				data = fmt.Sprintf("error: %v", res.err)
				msg = data
			} else {
				data = encode(res.data)
			}
			status := s.rdb.Set(context.Background(), res.id, data, 30*time.Minute)
			pubstatus := s.rdb.Publish(context.Background(), res.id, msg)
			s.logger.Info("work done", "id", res.id, "err", res.err, "set_err", status.Err(), "publish_err", pubstatus.Err())
		}
		//TODO: more than 1 worker?
		if count < 1 && len(queue) > 0 {
			count++
			next := queue[0]
			go generateSnapshot(next, done)
			queue = queue[1:]
			s.logger.Info("starting work", "id", next, "remaining", queue)
		}
	}
}

func generateSnapshot(id string, done chan resp) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(540, 250),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		opts...,
	)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	var buf []byte

	url := fmt.Sprintf("localhost:3001/%v", id)

	// capture entire browser viewport, returning png with quality=90
	err := chromedp.Run(ctx, fullScreenshot(url, 100, &buf))

	done <- resp{
		id:   id,
		data: buf,
		err:  err,
	}
}

func fullScreenshot(urlstr string, quality int, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(context.Context) error {
			return nil
		}),
		chromedp.Navigate(urlstr),
		chromedp.ActionFunc(func(context.Context) error {
			return nil
		}),
		chromedp.WaitEnabled("#status"),
		chromedp.ActionFunc(func(context.Context) error {
			//should handle checking if there's an error msgs here
			return nil
		}),
		chromedp.FullScreenshot(res, quality),
	}
}
