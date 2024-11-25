package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfig string
	target     string
	port       string
	certFile   string
	keyFile    string
	cAFile     string
)

func main() {

	flag.StringVar(&target, "target", "https://localhost:10250", "the https address")
	flag.StringVar(&port, "port", ":8039", "the http port")
	flag.StringVar(&kubeConfig, "kubeconfig", "/etc/kubernetes/kubeconfig/kubelet.kubeconfig", "the kubeconfig file")
	flag.StringVar(&certFile, "certFile", "", "the cert file")
	flag.StringVar(&keyFile, "keyFile", "", "the key file")
	flag.StringVar(&cAFile, "cAFile", "", "the ca file")

	flag.Parse()

	url, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	if kubeConfig != "" {
		loader := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig}
		loadedConfig, err := loader.Load()
		if err != nil {
			log.Fatal(err)
		}
		config, err := clientcmd.NewNonInteractiveClientConfig(
			*loadedConfig,
			loadedConfig.CurrentContext,
			&clientcmd.ConfigOverrides{},
			loader,
		).ClientConfig()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("use kubeconfig: %s", kubeConfig)
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(config.CAData)
		cliCrt, err := tls.X509KeyPair(config.CertData, config.KeyData)
		if err != nil {
			log.Fatalf("load x509 failed: %s, %v", certFile, err)
		}
		// 跳过 HTTPS 证书验证 (仅用于自签名证书)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				RootCAs:            pool,
				Certificates:       []tls.Certificate{cliCrt},
			},
		}
	} else if len(certFile) > 0 && len(keyFile) > 0 && len(cAFile) > 0 {
		pool := x509.NewCertPool()
		caData, err := ioutil.ReadFile(cAFile)
		if err != nil {
			log.Fatalf("read cAFile failed: %s, %v", cAFile, err)
		}
		pool.AppendCertsFromPEM(caData)
		cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatalf("load x509 failed: %s, %v", certFile, err)
		}

		// 跳过 HTTPS 证书验证 (仅用于自签名证书)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				RootCAs:            pool,
				Certificates:       []tls.Certificate{cliCrt},
			},
		}
	}

	// 配置 HTTP 服务器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request received: %s %s", r.Method, r.URL)
		r.URL.Scheme = "https"
		r.URL.Host = url.Host
		proxy.ServeHTTP(w, r)
	})

	// 启动 HTTP 服务器
	log.Printf("Starting HTTP to HTTPS reverse proxy server on %s%s\n", getIp(), port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func getIp() string {
	cmd := exec.Command("hostname", "-I")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.Fields(string(out))[0]
}
