打包

```shell
$env:GOOS="windows" 
$env:GOOS="linux" 
go build -trimpath -ldflags="-s -w" -o click .
```