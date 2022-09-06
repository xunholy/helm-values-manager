package main

import (
	"os"

	"github.com/xUnholy/helm-values-manager/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	f := cmd.NewRootCmd(os.Stdout, os.Args[1:])
	if err := f.Execute(); err != nil {
		os.Exit(1)
	}
}
