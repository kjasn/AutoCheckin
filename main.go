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

// Response 响应结构体
type Response struct {
	Success string `json:"success"`
	Msg     string `json:"msg"`
}

// Config 配置结构体
type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const (
	BASE = "./"
	LOG  = BASE + "dailyCheckin.log"

	// 第一个网站签到配置
	URL1       = "https://yc.yuchengyouxi.com/wp-admin/admin-ajax.php"
	METHOD1    = "POST"
	LOGIN_URL1 = "https://yc.yuchengyouxi.com/wp-login.php"

	// 第二个网站签到配置
	URL2       = "https://yxios.com/wp-admin/admin-ajax.php?action=checkin_details_modal"
	METHOD2    = "GET"
	LOGIN_URL2 = "https://yxios.com/wp-login.php"

	CONFIG_FILE = BASE + "config.json"
	MAX_RETRIES = 3
	RETRY_DELAY = 5 * time.Second
)

var (
	client *http.Client
	logger *log.Logger
	config Config
)

func init() {
	// 确保BASE目录存在
	if err := os.MkdirAll(BASE, 0666); err != nil {
		log.Printf("创建目录失败: %v", err)
	}

	// 设置日志
	logFile, err := os.OpenFile(LOG, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("无法创建日志文件: %v", err)
	} else {
		logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
	}

	// 读取配置文件
	configData, err := os.ReadFile(CONFIG_FILE)
	if err != nil {
		if logger != nil {
			logger.Fatal("无法读取配置文件:", err)
		} else {
			log.Fatal("无法读取配置文件:", err)
		}
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		if logger != nil {
			logger.Fatal("解析配置文件失败:", err)
		} else {
			log.Fatal("解析配置文件失败:", err)
		}
	}

	if config.Username == "" || config.Password == "" {
		if logger != nil {
			logger.Fatal("配置文件中缺少用户名或密码")
		} else {
			log.Fatal("配置文件中缺少用户名或密码")
		}
	}

	// 创建带cookie jar的HTTP客户端
	jar, err := cookiejar.New(nil)
	if err != nil {
		if logger != nil {
			logger.Fatal("创建cookie jar失败:", err)
		} else {
			log.Fatal("创建cookie jar失败:", err)
		}
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

	// 检查登录是否成功
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// signIn1 执行第一个网站的签到
func signIn1() (*Response, error) {
	data := url.Values{}
	data.Set("action", "daily_sign")

	req, err := http.NewRequest(METHOD1, URL1, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建签到请求失败: %v", err)
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

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// login2 执行第二个网站的登录获取cookie
func login2() error {
	data := url.Values{}
	data.Set("log", config.Username)
	data.Set("pwd", config.Password)
	data.Set("wp-submit", "登录")
	data.Set("redirect_to", "https://yxios.com/wp-admin/")
	data.Set("testcookie", "1")

	req, err := http.NewRequest("POST", LOGIN_URL2, bytes.NewBufferString(data.Encode()))
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

	return nil
}

// signIn2 执行第二个网站的签到
func signIn2() (*Response, error) {
	req, err := http.NewRequest(METHOD2, URL2, nil)
	if err != nil {
		return nil, fmt.Errorf("创建签到请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行签到请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 打印原始响应，帮助调试
	if logger != nil {
		logger.Printf("第二个网站原始响应: %s", string(body))
	} else {
		log.Printf("第二个网站原始响应: %s", string(body))
	}

	// 尝试解析为字符串类型的响应
	var resultStr string
	err = json.Unmarshal(body, &resultStr)
	if err == nil {
		return &Response{
			Success: "true",
			Msg:     resultStr,
		}, nil
	}

	// 尝试解析为数字类型的响应
	var resultNum int
	err = json.Unmarshal(body, &resultNum)
	if err == nil {
		return &Response{
			Success: "true",
			Msg:     fmt.Sprintf("签到返回数字: %d", resultNum),
		}, nil
	}

	// 如果以上解析都失败，返回原始响应
	return &Response{
		Success: "false",
		Msg:     string(body),
	}, nil
}

// retrySignIn1 带重试的第一个网站签到函数
func retrySignIn1() (*Response, error) {
	var lastErr error
	for i := 0; i < MAX_RETRIES; i++ {
		// 每次尝试前先登录刷新cookie
		if err := login1(); err != nil {
			if logger != nil {
				logger.Printf("第一个网站登录失败（第%d次尝试）: %v", i+1, err)
			} else {
				log.Printf("第一个网站登录失败（第%d次尝试）: %v", i+1, err)
			}
			lastErr = err
			time.Sleep(RETRY_DELAY)
			continue
		}

		result, err := signIn1()
		if err == nil {
			return result, nil
		}

		if logger != nil {
			logger.Printf("第一个网站签到失败（第%d次尝试）: %v", i+1, err)
		} else {
			log.Printf("第一个网站签到失败（第%d次尝试）: %v", i+1, err)
		}
		lastErr = err
		time.Sleep(RETRY_DELAY)
	}
	return nil, fmt.Errorf("达到最大重试次数，最后一次错误: %v", lastErr)
}

// retrySignIn2 带重试的第二个网站签到函数
func retrySignIn2() (*Response, error) {
	var lastErr error
	for i := 0; i < MAX_RETRIES; i++ {
		// 每次尝试前先登录刷新cookie
		if err := login2(); err != nil {
			if logger != nil {
				logger.Printf("第二个网站登录失败（第%d次尝试）: %v", i+1, err)
			} else {
				log.Printf("第二个网站登录失败（第%d次尝试）: %v", i+1, err)
			}
			lastErr = err
			time.Sleep(RETRY_DELAY)
			continue
		}

		result, err := signIn2()
		if err == nil {
			return result, nil
		}

		if logger != nil {
			logger.Printf("第二个网站签到失败（第%d次尝试）: %v", i+1, err)
		} else {
			log.Printf("第二个网站签到失败（第%d次尝试）: %v", i+1, err)
		}
		lastErr = err
		time.Sleep(RETRY_DELAY)
	}
	return nil, fmt.Errorf("达到最大重试次数，最后一次错误: %v", lastErr)
}

func main() {
	if logger != nil {
		logger.Println("=========================================")
		logger.Println("开始执行自动签到...")
	} else {
		logger.Println("=========================================")
		log.Println("开始执行自动签到...")
	}

	// 第一个网站签到
	result1, err1 := retrySignIn1()
	if err1 != nil {
		if logger != nil {
			logger.Printf("第一个网站签到失败: %v", err1)
		} else {
			log.Printf("第一个网站签到失败: %v", err1)
		}
	} else {
		if logger != nil {
			logger.Printf("第一个网站签到结果: Success=%s, Msg=%s", result1.Success, result1.Msg)
		} else {
			log.Printf("第一个网站签到结果: Success=%s, Msg=%s", result1.Success, result1.Msg)
		}
	}

	// 第二个网站签到
	result2, err2 := retrySignIn2()
	if err2 != nil {
		if logger != nil {
			logger.Printf("第二个网站签到失败: %v", err2)
		} else {
			log.Printf("第二个网站签到失败: %v", err2)
		}
	} else {
		if logger != nil {
			logger.Printf("第二个网站签到结果: Success=%s, Msg=%s", result2.Success, result2.Msg)
		} else {
			log.Printf("第二个网站签到结果: Success=%s, Msg=%s", result2.Success, result2.Msg)
		}
	}
}
