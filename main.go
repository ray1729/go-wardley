// This file is part of go-wardley.
//
// Copyright (C) 2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watch := flag.Bool("watch", false, "Watch input for changes")
	input := flag.String("input", "", "Input file or directory")
	output := flag.String("output", "", "Output file or directory")
	flag.Parse()

	if *input == "" {
		log.Fatal("--input is required")
	}

	if !strings.HasSuffix(*input, ".hcl") {
		log.Fatal("Expected a .hcl input file")
	}

	if *output == "" {
		*output = (*input)[:len(*input)-4] + ".svg"
	}

	err := render(*input, *output)
	if err != nil {
		log.Println(err)
	}

	if !*watch {
		return

	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = watcher.Add(*input)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				break
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if err := render(*input, *output); err != nil {
					log.Println(err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				break
			}
			log.Println("Error: ", err)
		}
	}
}

func render(infile, outfile string) error {
	inInfo, err := os.Stat(infile)
	if err != nil {
		return err
	}
	outInfo, err := os.Stat(outfile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && outInfo.ModTime().After(inInfo.ModTime()) {
		return fmt.Errorf("%s exists and is newer than %s", outfile, infile)
	}
	log.Printf("Rendering %s to %s", infile, outfile)
	m, err := ParseFile(infile)
	if err != nil {
		return err
	}
	ofh, err := os.Create(outfile)
	if err != nil {
		return err
	}
	m.RenderSVG(ofh)
	return ofh.Close()
}
