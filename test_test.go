package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"gjt/slog"
	"log"
	"math"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func TestJt(t *testing.T) {
	Screenshot("http://54.154.145.128")
}

func HttpGet(url string) int {

	timeout := time.Duration(10 * time.Second)

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)

	if err != nil {
		slog.Println(slog.DEBUG, err)
		return 0
	}
	defer resp.Body.Close()

	// slog.Println(slog.DEBUG, url, "status", resp.StatusCode)
	return resp.StatusCode
}

var ScrenCount int32 = 0

func Screenshot(url string) {

	atomic.AddInt32(&ScrenCount, 1)

	defer atomic.AddInt32(&ScrenCount, -1)

	if HttpGet(url) != 200 {
		// slog.Println(slog.DEBUG, url, "不可访问")
		return
	}

	slog.Println(slog.DEBUG, url, "开始截图")

	// 禁用chrome headless
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck, //不检查默认浏览器
		chromedp.Flag("headless", true),
		chromedp.Flag("blink-settings", "imagesEnabled=true"), //开启图像界面,重点是开启这个
		chromedp.Flag("ignore-certificate-errors", true),      //忽略错误
		chromedp.Flag("disable-web-security", true),           //禁用网络安全标志
		chromedp.Flag("disable-extensions", true),             //开启插件支持
		chromedp.Flag("disable-default-apps", true),
		chromedp.WindowSize(1920, 1080),    // 设置浏览器分辨率（窗口大小）
		chromedp.Flag("disable-gpu", true), //开启gpu渲染
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.NoFirstRun, //设置网站不是首次运行
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36"), //设置UserAgent
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 创建上下文实例
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// 创建超时上下文
	ctx, cancel = context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	//导航到目标页面，等待一个元素，捕捉元素的截图
	var buf []byte
	// capture entire browser viewport, returning png with quality=90
	if err := chromedp.Run(ctx, fullScreenshot(url, 100, &buf)); err != nil {
		slog.Println(slog.DEBUG, err)
	}
	//sslog.Println(slog.DEBUG,slog.DEBUG, url)
	path := Md5(url) + ".png"
	WritePng(path, buf)

}

// 获取整个浏览器窗口的截图（全屏）
// 这将模拟浏览器操作设置。
func fullScreenshot(urlstr string, quality int64, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		//chromedp.WaitVisible("style"),
		chromedp.Sleep(10 * time.Second),
		//chromedp.OuterHTML(`document.querySelector("body")`, &htmlContent, chromedp.ByJSPath),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 得到布局页面
			_, _, _, _, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}

			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

			// 浏览器视窗设置模拟
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}

			// 捕捉屏幕截图
			*res, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  contentSize.Width,
					Height: contentSize.Height,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
}

// 获取截图路径
func GetScreenPath() string {
	return "/u2/cyberspace/www/data-clean/public/screen/"
}

func WritePng(name string, buf []byte) {
	//sslog.Println(slog.DEBUG,slog.WARN, path)
	_, err := os.Stat(GetScreenPath())
	if err != nil {
		os.MkdirAll(GetScreenPath(), 0777)
	}

	f, err := os.OpenFile(GetScreenPath()+name, os.O_CREATE+os.O_RDWR, 0664)
	if err != nil {
		slog.Println(slog.DEBUG, err)
		return
	}

	f.Write(buf)

	//sslog.Println(slog.DEBUG,slog.DEBUG, "图片写入完成")
}

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
