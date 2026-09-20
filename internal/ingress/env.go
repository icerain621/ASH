package ingress

import "os"

func stdGetenv(k string) string { return os.Getenv(k) }
