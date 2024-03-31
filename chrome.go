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
			if res.err != nil {
				status := s.rdb.Set(context.Background(), res.id, res.err.Error(), 5*time.Second)
				pubstatus := s.rdb.Publish(context.Background(), res.id, res.err.Error())
				s.logger.Info("work failed", "id", res.id, "err", res.err, "set_err", status.Err(), "publish_err", pubstatus.Err())
			} else {
				status := s.rdb.Set(context.Background(), res.id, encode(res.data), s.cacheTTL)
				pubstatus := s.rdb.Publish(context.Background(), res.id, "done")
				s.logger.Info("work done", "id", res.id, "err", res.err, "set_err", status.Err(), "publish_err", pubstatus.Err())
			}
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
	res, err := s.generateSnapshot(s.previewURL + "/" + id)
	s.logger.Info("work completed", "err", err)
	done <- resp{
		id:   id,
		data: res,
		err:  err,
	}
}

func (s *Server) generateSnapshot(url string) ([]byte, error) {
	browser := rod.New().Client(s.l.MustClient())
	err := browser.Connect()
	if err != nil {
		return nil, fmt.Errorf("error connecting to browser: %w", err)
	}
	log.Println("browser connect ok")
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("error creating page: %w", err)
	}
	_, err = page.SetExtraHeaders([]string{"X-CUSTOM-AUTH-KEY", s.authKey})
	if err != nil {
		return nil, fmt.Errorf("error setting extra headers: %w", err)
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

	_, err = page.Race().ElementFunc(func(p *rod.Page) (*rod.Element, error) {
		res, err := page.Evaluate(rod.Eval(`(s, n) => document.querySelectorAll(s).length > n`, "div", 10))
		if err != nil {
			return nil, &rod.ElementNotFoundError{}
		}
		if res.Value.Bool() {
			return &rod.Element{}, nil
		}

		return nil, &rod.ElementNotFoundError{}
	}).Element("#has-error").Handle(func(e *rod.Element) error {
		str, err := e.Attribute("value")
		// can't do much aobut this err here other than log it
		if err != nil {
			s.logger.Info("error encountered looking for value attribute", "err", err)
			return fmt.Errorf("unexpected server error: %v", err)
		}
		return fmt.Errorf("generate preview failed: %v", *str)
	}).Do()

	if err != nil {
		return nil, err
	}

	log.Println("race done")

	buf, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatWebp,
		Quality: gson.Int(100),
	})
	return buf, err
}
