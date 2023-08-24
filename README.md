# lilly

![lilly](lilly.png)
#### meow-meow WebSocket server

## export & proto build

```bash
export PATH="/Users/{user_go_path}/go/bin:$PATH"
protoc -I . --go_out=./proto --go-grpc_out=./proto ./proto/*.proto
```
