package preview

import (
	context "context"
	"log"

	"github.com/chromedp/chromedp"
)

func generateSnapshot(url string) ([]byte, error) {
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

	// capture entire browser viewport, returning png with quality=90
	if err := chromedp.Run(ctx, fullScreenshot(url, 100, &buf)); err != nil {
		return nil, err
	}

	return buf, nil
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
		chromedp.WaitEnabled("#card"),
		chromedp.ActionFunc(func(context.Context) error {
			//should handle checking if there's an error msgs here
			return nil
		}),
		chromedp.FullScreenshot(res, quality),
	}
}
