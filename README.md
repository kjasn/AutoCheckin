# 每日签到脚本

## 使用方法

将本项目 fork 到自己的仓库，添加或修改仓库的 secrets variables，然后修改 .github/workflows/daily_checkin.yml 中的 cron 时间，然后每天定时执行即可。

## 注意事项

- 由于是自动获取 cookie，所以如果登录需要验证码，则需要手动登录，然后获取 cookie。
- 不同网站的登录和签到逻辑不同，所以需要根据实际情况修改 main.go 中的代码。
