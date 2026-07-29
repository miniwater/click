打包

```shell
$env:GOOS="windows" 
$env:GOOS="linux" 
go clean -cache
go build -trimpath -ldflags="-s -w" -o click .
```