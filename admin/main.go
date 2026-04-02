package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "FunnyOption admin Go runtime is deprecated and no longer starts a service.")
	fmt.Fprintln(os.Stderr, "Use the single supported Next.js admin runtime instead:")
	fmt.Fprintln(os.Stderr, "  /Users/zhangza/code/funnyoption/scripts/dev-up.sh")
	fmt.Fprintln(os.Stderr, "or")
	fmt.Fprintln(os.Stderr, "  cd /Users/zhangza/code/funnyoption/admin && npm run dev -- --hostname 127.0.0.1 --port 3001")
	os.Exit(1)
}
