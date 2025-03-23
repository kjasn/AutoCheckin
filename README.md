# 每日签到脚本

🤖 使用 Github Actions 实现每日自动签到，并随机延迟时间(0~10min)。

## 使用方法

1. 将本项目 fork 到自己的仓库
2. 添加或修改仓库的 secrets variables (USERNAME, PASSWORD 默认多个站点账号密码用同一个，请按需修改)
3. 然后按需修改 `.github/workflows/daily_checkin.yml` 中的 cron 时间。
    > cron 时间是 UTC 时间，需要根据自己的时区调整。
    > 以北京时间（UTC + 8 ） 为例，实际执行时间为设置的 cron 时间 + 8 小时。

## 注意事项

-   由于是自动获取 cookie，所以如果登录需要验证码，则需要手动登录，然后获取 cookie。
-   不同网站的登录和签到逻辑不同，所以需要根据实际情况修改 main.go 中的代码。
