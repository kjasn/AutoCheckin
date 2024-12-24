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
)

// FirstResponse 表示第一个网站的响应结构体
type FirstResponse struct {
	Success string `json:"success"`
	Msg     string `json:"msg"`
}

/*
// SecondResponse 表示第二个网站的响应结构体
type SecondResponse struct {
	Msg  string `json:"msg"`
	Data struct {
		Integral int    `json:"integral"`
		Points   int    `json:"points"`
		Time     string `json:"time"`
	} `json:"data"`
	ContinuousDay int  `json:"continuous_day"`
	Error         bool `json:"error"`
}
*/

// Config 配置结构体
type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const (
	BASE = "./"
	// LOG  = BASE + "dailyCheckin.log"

	// 第一个网站签到配置
	URL1       = "https://yc.yuchengyouxi.com/wp-admin/admin-ajax.php"
	METHOD1    = "POST"
	LOGIN_URL1 = "https://yc.yuchengyouxi.com/wp-login.php"

	/*
		// 第二个网站签到配置
		// URL2       = "https://yxios.com/wp-admin/admin-ajax.php?action=checkin_details_modal"
		URL2       = "https://yxios.com/wp-admin/admin-ajax.php"
		METHOD2    = "POST"
		LOGIN_URL2 = "https://yxios.com/user-sign?tab=signin&redirect_to=https%3A%2F%2Fyxios.com%2Fuser%2Faccount"
	*/

	CONFIG_FILE = BASE + "config.json"
	MAX_RETRIES = 3
	RETRY_DELAY = 5 * time.Second
)

var (
	client *http.Client
	// logger *log.Logger
	config       Config
	savedCookies []*http.Cookie
)

func init() {
	// 读取配置文件
	configData, err := os.ReadFile(CONFIG_FILE)
	if err != nil {
		log.Fatal("无法读取配置文件:", err)
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatal("解析配置文件失败:", err)
	}

	if config.Username == "" || config.Password == "" {
		log.Fatal("配置文件中缺少用户名或密码")
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

// login1 执行第一个网站的登录获取cookie
func login1() error {
	data := url.Values{}
	data.Set("log", config.Username)
	data.Set("pwd", config.Password)
	data.Set("wp-submit", "登录")
	data.Set("redirect_to", "https://yc.yuchengyouxi.com/wp-admin/")
	data.Set("testcookie", "1")

	req, err := http.NewRequest("POST", LOGIN_URL1, bytes.NewBufferString(data.Encode()))
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
	log.Printf("登录响应: %#v", resp)

	// 检查登录是否成功
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败，状态码: %d", resp.StatusCode)
	}

	// 登录成功后，保存 cookies
	savedCookies = resp.Cookies() // 直接保存响应中的 cookies

	return nil
}

// checkIn1 执行第一个网站的签到
func checkIn1() (*FirstResponse, error) {
	data := url.Values{}
	data.Set("action", "daily_sign")

	req, err := http.NewRequest(METHOD1, URL1, bytes.NewBufferString(data.Encode()))
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
	log.Printf("签到响应: %#v", resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result FirstResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// retryCheckIn1 带重试的第一个网站签到函数
func retryCheckIn1() (*FirstResponse, error) {
	var lastErr error
	for i := 0; i < MAX_RETRIES; i++ {
		// 每次尝试前先登录刷新cookie
		if err := login1(); err != nil {
			log.Printf("第一个网站登录失败（第%d次尝试）: %v", i+1, err)
			lastErr = err
			time.Sleep(RETRY_DELAY)
			continue
		}

		// 打印登录成功后的状态
		log.Println("第一个网站登录成功，准备进行签到...")

		result, err := checkIn1()
		if err == nil {
			return result, nil
		}

		log.Printf("第一个网站签到失败（第%d次尝试）: %v", i+1, err)
		lastErr = err
		time.Sleep(RETRY_DELAY)
	}
	return nil, fmt.Errorf("达到最大重试次数，最后一次错误: %v", lastErr)
}

func main() {
	log.Println("=========================================")
	log.Println("开始执行自动签到...")

	// 第一个网站签到
	result1, err1 := retryCheckIn1()
	if err1 != nil {
		log.Printf("第一个网站签到失败: %v", err1)
	} else {
		log.Printf("第一个网站签到结果: Success=%s, Msg=%s", result1.Success, result1.Msg)
	}
}
