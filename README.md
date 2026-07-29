打包

```shell
$env:GOOS="linux" 
go build -trimpath -ldflags="-s -w" -o click .
```