package main

import (
	"flag"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

var help = flag.Bool("help", false, "show help info")
var version = flag.Bool("version", false, "show version info")

func main() {
	flag.Parse()
	if *help || *version {
		fmt.Println("tiktoken example.txt")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("tiktoken example.txt")
		os.Exit(0)
	}
	file, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	text := string(file)
	if !utf8.ValidString(text) {
		fmt.Println("invalid utf8 text")
		os.Exit(1)
	}

	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	tk, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s %d tokens\n", args[0], len(tk.Encode(text, nil, nil)))
	os.Exit(0)
}
