package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
)

var (
	fcolor string
	format string
	filter string
)

func args2config() (tail.Config, int64) {
	config := tail.Config{Follow: true}

	n := int64(0)
	maxlinesize := int(0)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s [OPTIONS] file:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&config.Follow, "f", false, "wait for additional data to be appended to the file")
	flag.BoolVar(&config.Poll, "p", false, "use polling, instead of inotify")
	flag.BoolVar(&config.ReOpen, "F", false, "follow, and track file rename/rotation")
	flag.Int64Var(&n, "n", 0, "tail from the last Nth location")
	flag.IntVar(&maxlinesize, "max", 0, "max line size")
	flag.StringVar(&fcolor, "color", "", "Comma separate coloring output (default color: [1: red 2: green 3: yellow 4: blue 5: magenta 6: cyan])")
	flag.StringVar(&filter, "filter", "", "Comma separate filtering output")
	flag.StringVar(&format, "format", "plain", "Output format [\"plain\", \"json\"]")
	flag.Parse()
	if config.ReOpen {
		config.Follow = true
	}
	config.MaxLineSize = maxlinesize
	if runtime.GOOS == "darwin" {
		config.Poll = true
	}
	return config, n
}

func main() {
	config, n := args2config()
	if flag.NFlag() < 1 {
		fmt.Println("need one or more files as arguments")
		os.Exit(1)
	}

	if n != 0 {
		config.Location = &tail.SeekInfo{-n, os.SEEK_END}
	}

	done := make(chan bool)
	for _, filename := range flag.Args() {
		go tailFile(filename, config, done)
	}

	for _, _ = range flag.Args() {
		<-done
	}
}

func tailFile(filename string, config tail.Config, done chan bool) {
	defer func() { done <- true }()
	t, err := tail.TailFile(filename, config)
	if err != nil {
		fmt.Println("%s", err)
		return
	}

	for l := range t.Lines {
		switch format {
		case "plain":
			if filter != "" {
				fsplit := strings.Split(filter, ",")
				for _, f := range fsplit {
					if strings.Index(l.Text, f) == -1 {
						continue
					}
				}
			}

			if fcolor != "" {
				csplit := strings.Split(fcolor, ",")
				for i, c := range csplit {
					if strings.Index(l.Text, c) > -1 {
						switch i {
						case 0:
							fmt.Println(color.RedString(l.Text))
						case 1:
							fmt.Println(color.GreenString(l.Text))
						case 2:
							fmt.Println(color.YellowString(l.Text))
						case 3:
							fmt.Println(color.BlueString(l.Text))
						case 4:
							fmt.Println(color.MagentaString(l.Text))
						case 5:
							fmt.Println(color.CyanString(l.Text))
						}
					}
				}
			} else {
				fmt.Println(l.Text)
			}

		case "json":
			if len(l.Text) > 0 {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(l.Text), &data); err != nil {
					continue
				}
				jdata, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					fmt.Println("%s", err)
				}
				os.Stdout.Write(append(jdata, '\n'))
			}

		default:
			fmt.Errorf("Unknown format")
		}
	}
	err = t.Wait()
	if err != nil {
		fmt.Println("%s", err)
	}
}

func jsonFormat() {
	if len(os.Args) != 2 {
		fmt.Println("One argument, the json file to pretty-print is required")
		os.Exit(-1)
	}

	fileName := os.Args[1]
	byt, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	var dat map[string]interface{}

	if err := json.Unmarshal(byt, &dat); err != nil {
		panic(err)
	}
	b, err := json.MarshalIndent(dat, "", "  ")
	if err != nil {
		panic(err)
	}
	b2 := append(b, '\n')
	os.Stdout.Write(b2)

}
