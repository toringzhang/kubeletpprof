# kubeletpprof

原生go tools不支持pprof https接口，本项目将kubelet 10250https接口转为http接口，便于使用pprof

## 使用方式

1. 使用kubeconfig文件

```bash
./http2https -kubeconfig /etc/kubernetes/kubeconfig/kubelet.kubeconfig
```

2. 使用ca证书

```bash
./http2https -cAFile ca.pem -certFile apiserver-client.pem -keyFile apiserver-client-key.pem
```

3. 执行go tools

```bash
go tool pprof -http :8080 http://10.79.74.36:8039/debug/pprof/heap
```
