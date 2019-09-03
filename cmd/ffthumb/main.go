// Example command line program using ffthumb.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"text/tabwriter"

	"github.com/7fffffff/ffthumb"
)

var numThumbnails int
var numWorkers int

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}

func main() {
	fs := flag.NewFlagSet("ffthumb", flag.ExitOnError)
	fs.IntVar(&numThumbnails, "n", 5, "number of thumbnails to generate and select from")
	fs.IntVar(&numWorkers, "w", 1, "number of workers")
	fs.Usage = usageFor(fs, "ffthumb [flags] <paths to video files>")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	filePaths := fs.Args()
	if len(filePaths) < 1 {
		fs.Usage()
		os.Exit(1)
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, os.Kill)
		<-stop
		cancelFn()
	}()
	if numWorkers < 1 {
		numWorkers = 1
	}
	wg := &sync.WaitGroup{}
	work := make(chan string, len(filePaths))
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pickWorker(ctx, wg, work, numThumbnails)
	}
	for _, filePath := range filePaths {
		work <- filePath
	}
	close(work)
	wg.Wait()
}

func pickWorker(ctx context.Context, wg *sync.WaitGroup, input chan string, n int) {
	defer wg.Done()
	for {
		select {
		case w, ok := <-input:
			if !ok {
				return
			}
			err := pick(ctx, w, n)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func pick(ctx context.Context, inputPath string, n int) error {
	if n < 1 {
		n = 1
	}
	outputPath := inputPath + ".png"
	thumbnailer := &ffthumb.Thumbnailer{Num: n}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(file.Name())
		}
	}()
	err = thumbnailer.WriteThumbnail(ctx, file, inputPath)
	if err != nil {
		return err
	}
	fmt.Println(outputPath)
	return nil
}
