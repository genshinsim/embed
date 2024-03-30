package preview

import (
	context "context"
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
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
			go s.doWork(next, done)
			queue = queue[1:]
			s.logger.Info("starting work", "id", next, "remaining", queue)
		}
	}
}

func (s *Server) doWork(id string, done chan resp) {
	res, err := generateSnapshot("http://localhost:3001/" + id)
	s.logger.Info("work completed", "err", err)
	done <- resp{
		id:   id,
		data: res,
		err:  err,
	}
}

func generateSnapshot(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	browser := rod.New()
	browser.Context(ctx)
	err := browser.Connect()
	if err != nil {
		return nil, fmt.Errorf("error connecting to browser: %w", err)
	}
	log.Println("browser connect ok")
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("error creating page: %w", err)
	}
	log.Println("page load ok")
	page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  520,
		Height: 250,
	})
	err = page.Navigate(url)
	if err != nil {
		return nil, fmt.Errorf("error navigating to page: %w", err)
	}
	log.Println("navigated to page")

	page.Race().ElementFunc(func(p *rod.Page) (*rod.Element, error) {
		log.Println("js eval started")
		res, err := page.Evaluate(rod.Eval(`(s, n) => document.querySelectorAll(s).length > n`, "div", 10))
		if err != nil {
			log.Println("eval failed? ", err)
			return nil, &rod.ElementNotFoundError{}
		}
		log.Println("js eval done", res)
		if res.Value.Bool() {
			log.Println("loaded ok")
			return nil, nil
		}

		return nil, &rod.ElementNotFoundError{}
	}).Element("#has-error").Handle(func(e *rod.Element) error {
		log.Println("error found")
		return nil
	}).Do()

	log.Println("race done")

	buf, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatWebp,
		Quality: gson.Int(100),
	})
	return buf, err
}

// func fullScreenshot(urlstr string, quality int, res *[]byte) chromedp.Tasks {
// 	return chromedp.Tasks{
// 		chromedp.ActionFunc(func(context.Context) error {
// 			log.Println("hello?")
// 			return nil
// 		}),
// 		chromedp.Navigate(urlstr),
// 		chromedp.ActionFunc(func(context.Context) error {
// 			log.Println("navigated to ", urlstr)
// 			return nil
// 		}),
// 		chromedp.WaitEnabled("#status"),
// 		chromedp.ActionFunc(func(context.Context) error {
// 			//should handle checking if there's an error msgs here
// 			log.Println("status ok?")
// 			return nil
// 		}),
// 		chromedp.FullScreenshot(res, quality),
// 	}
// }
