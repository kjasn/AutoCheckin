package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// 签到任务响应结构体
type TaskResponse struct {
	Success string `json:"success"`
	Msg     string `json:"msg"`
}

const (
	MAX_RETRIES = 3
	RETRY_DELAY = 5 * time.Second
)

// 站点配置结构体
type SiteConfig struct {
	BaseURL    string
	LoginURL   string
	CheckInURL string
	Method     string
}

// 站点配置
var sites = []SiteConfig{
	{
		BaseURL:    "https://yc.yuchengyouxi.com/",
		LoginURL:   "https://yc.yuchengyouxi.com/wp-login.php",
		CheckInURL: "https://yc.yuchengyouxi.com/wp-admin/admin-ajax.php",
		Method:     "POST",
	},
	{
		BaseURL:    "https://ios.liferm.com/",
		LoginURL:   "https://ios.liferm.com/wp-login.php",
		CheckInURL: "https://ios.liferm.com/wp-admin/admin-ajax.php",
		Method:     "POST",
	},
}

var (
	client       *http.Client
	savedCookies []*http.Cookie
)

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Fatal("加载配置文件(.env) 失败:", err)
	}

	// 创建带cookie jar的HTTP客户端
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("创建cookie jar失败:", err)
	}
	client = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

// login 登录网站，获取cookie
func login(site SiteConfig) error {
	USERNAME := os.Getenv("USERNAME")
	PASSWORD := os.Getenv("PASSWORD")
	data := url.Values{}
	data.Set("log", USERNAME)
	data.Set("pwd", PASSWORD)
	data.Set("wp-submit", "登录")
	data.Set("redirect_to", site.BaseURL+"wp-admin/")
	data.Set("testcookie", "1")

	req, err := http.NewRequest("POST", site.LoginURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("创建登录请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("执行登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查登录是否成功
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败，状态码: %d", resp.StatusCode)
	}

	// 登录成功后，保存 cookies
	savedCookies = resp.Cookies() // 直接保存响应中的 cookies

	return nil
}

// checkIn 签到
func checkIn(site SiteConfig) (*TaskResponse, error) {
	data := url.Values{}
	data.Set("action", "daily_sign")

	req, err := http.NewRequest(site.Method, site.CheckInURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建签到请求失败: %v", err)
	}

	for _, cookie := range savedCookies {
		req.AddCookie(cookie)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行签到请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result TaskResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// retryCheckIn 带重试的网站签到函数
func retryCheckIn(site SiteConfig) (*TaskResponse, error) {
	var lastErr error
	for i := 0; i < MAX_RETRIES; i++ {
		// 每次尝试前先登录刷新cookie
		if err := login(site); err != nil {
			log.Printf("网站 %s 登录失败（第%d次尝试）: %v", site.BaseURL, i+1, err)
			lastErr = err
			time.Sleep(RETRY_DELAY)
			continue
		}

		// 打印登录成功后的状态
		log.Printf("网站 %s 登录成功，准备进行签到...", site.BaseURL)

		result, err := checkIn(site)
		if err == nil {
			return result, nil
		}

		log.Printf("网站 %s 签到失败（第%d次尝试）: %v", site.BaseURL, i+1, err)
		lastErr = err
		time.Sleep(RETRY_DELAY)
	}
	return nil, fmt.Errorf("达到最大重试次数，最后一次错误: %v", lastErr)
}

func main() {
	log.Println("=========================================")
	log.Println("开始执行自动签到...")

	for _, site := range sites {
		log.Println("-----------------------------------------")
		result, err := retryCheckIn(site)
		if err != nil {
			log.Printf("网站 %s 签到失败: %v", site.BaseURL, err)
		} else {
			log.Printf("网站 %s 签到结果: Success=%s, Msg=%s", site.BaseURL, result.Success, result.Msg)
		}
	}
}
