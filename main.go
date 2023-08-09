package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type AutoGenerated struct {
	Body        string `json:"Body"`
	CreateDate  string `json:"CreateDate"`
	CreateTime  string `json:"CreateTime"`
	Digest      string `json:"Digest"`
	Domain      string `json:"Domain"`
	FingerPrint string `json:"FingerPrint"`
	FoundDomain string `json:"FoundDomain"`
	Header      string `json:"Header"`
	IP          string `json:"IP"`
	Icon        string `json:"Icon"`
	Length      string `json:"Length"`
	Port        string `json:"Port"`
	Response    string `json:"Response"`
	Service     string `json:"Service"`
}

const ScanLimit = 10

var ScrenCount int32 = 0

func main() {
	log.Println("stsrt")
	xfiles, _ := GetFiles("/u2/zrtx/log/cyberspace", "ipInfo")

	fmt.Println(xfiles)

	scanedFilePath := "scanedFile.txt"
	scanDF, _ := ReadLineData(scanedFilePath)

	for _, xfile := range xfiles {

		log.Println("file", xfile)

		if In_array(xfile, scanDF) {
			log.Println("扫过file", xfile)
			continue
		}

		datas, _ := ReadLineData(xfile)

		lastLinePath := "lastLine.txt"

		lastLine := Read(lastLinePath)

		lineLast, _ := strconv.Atoi(lastLine)

		for line, data := range datas {

			if line <= lineLast {
				log.Println("扫过line", line)
				continue
			}

			var dataJson AutoGenerated

			err := json.Unmarshal([]byte(data), &dataJson)
			if err != nil {
				log.Println(err)
				continue
			}

			if dataJson.Service != "http" || dataJson.Service != "https" {
				continue
			}

			//非200  不截图
			if !strings.Contains(dataJson.Header, "200") {
				continue
			}

			for ScrenCount > ScanLimit {
				time.Sleep(1 * time.Second)
			}

			if dataJson.Domain != "" {
				go Screenshot(dataJson.Service + "://" + dataJson.Domain)
			} else {
				go Screenshot(dataJson.Service + "://" + dataJson.IP + ":" + dataJson.Port)
			}

			Write(lastLinePath, GetInterfaceToString(line))
		}

		Write(lastLinePath, "0")
		WriteAppend(scanedFilePath, xfile)
	}
}

func Screenshot(url string) {
	atomic.AddInt32(&ScrenCount, 1)
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
		log.Println(err)
	}
	//slog.Println(slog.DEBUG, url)
	path := Md5(url) + ".png"
	WritePng(path, buf)

	atomic.AddInt32(&ScrenCount, -1)
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
	//slog.Println(slog.WARN, path)
	_, err := os.Stat(GetScreenPath())
	if err != nil {
		os.MkdirAll(GetScreenPath(), 0777)
	}

	f, err := os.OpenFile(GetScreenPath()+name, os.O_CREATE+os.O_RDWR, 0664)
	if err != nil {
		log.Println(err)
		return
	}

	f.Write(buf)

	//slog.Println(slog.DEBUG, "图片写入完成")
}

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func GetFiles(root, name string) (files []string, err error) {

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, name) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// 换行的数据
func ReadLineData(userDict string) (users []string, err error) {
	file, err := os.Open(userDict)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		user := strings.TrimSpace(scanner.Text())
		if user != "" {
			users = append(users, user)
		}
	}
	return users, err
}

func In_array(needle interface{}, hystack interface{}) bool {
	switch key := needle.(type) {
	case string:
		for _, item := range hystack.([]string) {
			if key == item {
				return true
			}
		}
	case int:
		for _, item := range hystack.([]int) {
			if key == item {
				return true
			}
		}
	case int64:
		for _, item := range hystack.([]int64) {
			if key == item {
				return true
			}
		}
	default:
		return false
	}
	return false
}

func Read(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Println("router.Run error", err)
	}
	return string(content)
}

// 任意类型转为str
func GetInterfaceToString(value interface{}) string {
	// interface 转 string
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}

func Write(path, str string) {
	f, err := os.OpenFile(path, os.O_CREATE+os.O_RDWR+os.O_TRUNC, 0764)
	if err != nil {
		log.Panicln("router.Run error", err)
	}

	//jsonBuf := append([]byte(result),[]byte("\r\n")...)
	f.WriteString(str)
}

func WriteAppend(path, str string) {
	f, err := os.OpenFile(path, os.O_CREATE+os.O_RDWR+os.O_APPEND, 0764)
	if err != nil {
		log.Fatal(err)
	}

	//jsonBuf := append([]byte(result),[]byte("\r\n")...)
	str += "\n"
	f.WriteString(str)
}
