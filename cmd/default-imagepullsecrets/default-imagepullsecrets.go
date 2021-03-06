package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gjkim42/default-imagepullsecrets/pkg/admission"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
)

func main() {

	checkErr(os.Stderr, NewDefaultImagePullSecretsCommand().Execute())
}

func checkErr(w io.Writer, err error) {
	if err != nil {
		fmt.Fprintln(w, err)
		os.Exit(1)
	}
}

func NewDefaultImagePullSecretsCommand() *cobra.Command {
	options := &DefaultImagePullSecretsOptions{
		log: klogr.New(),
	}
	certFile := "tls.crt"
	keyFile := "tls.key"
	bindAddress := "0.0.0.0"
	port := 443
	cmd := &cobra.Command{
		Use:   "default-imagepullsecrets",
		Short: "The admission controller that applies default imagePullSecrets to pods",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(os.Stderr, options.Complete(certFile, keyFile, bindAddress, port))
			checkErr(os.Stderr, options.Run())
		},
	}

	klog.InitFlags(nil)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().StringVar(&certFile, "cert-file", certFile, "File containing the default Certificate for HTTPS.")
	cmd.Flags().StringVar(&keyFile, "key-file", keyFile, "File containing the default Key for HTTPS.")
	cmd.Flags().StringVar(&bindAddress, "bind-address", bindAddress, "The address on which to listen for the webhook's server")
	cmd.Flags().IntVar(&port, "port", port, "The port on which to serve the webhook's server")
	cmd.Flags().StringSliceVar(&options.ImagePullSecrets, "image-pull-secrets", options.ImagePullSecrets, "ImagePullSecrets to be added to each pod")

	return cmd
}

type DefaultImagePullSecretsOptions struct {
	log logr.Logger

	Address   string
	TLSConfig *tls.Config

	ImagePullSecrets []string
}

func (o *DefaultImagePullSecretsOptions) Complete(certFile, keyFile, bindAddress string, port int) error {
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
		o.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	o.Address = fmt.Sprintf("%s:%d", bindAddress, port)

	return nil
}

func (o *DefaultImagePullSecretsOptions) Run() error {
	http.Handle("/webhook", admission.NewController(o.log.WithName("admission"), o.ImagePullSecrets))
	http.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("ok")) })

	server := &http.Server{
		Addr:      o.Address,
		TLSConfig: o.TLSConfig,
	}

	return server.ListenAndServeTLS("", "")
}
