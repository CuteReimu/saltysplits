# SaltySplits

这是一个用于分析 LiveSplits 的分段文件（.lss）的工具。它可以帮助用户更好地理解和管理他们的分段数据。

其灵感来自于：[jaspersiebring/saltysplits](https://github.com/jaspersiebring/saltysplits)

## 开发相关

由于依赖了很多第三方库，所以首先需要运行 `dowload_cdn.sh` 或 `dowload_cdn.ps1` 脚本来下载这些第三方库。

之后使用 Go 编译即可，本人开发用的是 Go 1.25，但估计更低的版本也能使用，没有太多兼容问题。

```bash
go build -o saltysplits.exe
```

## 使用的第三方库

NodeJs库：
- Vue
- ChartJs
- Element Plus
- Axios

Go库：
- Gin