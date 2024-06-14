package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/fatih/color"
)

const numWorkers = 4 // Number of worker goroutines

var (
	successLogger = log.New(os.Stdout, "", log.LstdFlags)
	errorLogger   = log.New(os.Stderr, "", log.LstdFlags)
	skipLogger    = log.New(os.Stdout, "", log.LstdFlags)
)

func main() {
	// Check if ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		errorLogger.Println(color.RedString("FFMPEG not found"))
		return
	}

	// Check if a file path is provided
	if len(os.Args) < 2 {
		errorLogger.Println(color.RedString("Please provide the path to an MKV file or a directory"))
		return
	}

	// Get the file path from the command line arguments
	inputPath := os.Args[1]

	// Channel to collect .mkv files to process
	filesChan := make(chan string, 100)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(filesChan, &wg)
	}

	// Check if the provided path is a directory
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		errorLogger.Printf(color.RedString("Error accessing the path: %v", err))
		close(filesChan)
		wg.Wait()
		return
	}

	if fileInfo.IsDir() {
		// If it's a directory, walk through it and collect all .mkv files
		err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errorLogger.Printf(color.RedString("Error accessing file %s: %v", path, err))
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".mkv" {
				filesChan <- path
			}
			return nil
		})
		if err != nil {
			errorLogger.Printf(color.RedString("Error walking through the directory: %v", err))
		}
	} else {
		// If it's a file, process it if it's an .mkv file
		if filepath.Ext(inputPath) == ".mkv" {
			filesChan <- inputPath
		} else {
			errorLogger.Println(color.RedString("The provided file is not an MKV file"))
		}
	}

	// Close the channel and wait for all workers to finish
	close(filesChan)
	wg.Wait()
}

func worker(filesChan chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range filesChan {
		processMKVFile(filePath)
	}
}

func processMKVFile(filePath string) {
	// Construct the output file path
	outputFilePath := filePath[:len(filePath)-len(filepath.Ext(filePath))] + ".vtt"

	// Check if the .vtt file already exists
	if _, err := os.Stat(outputFilePath); err == nil {
		skipLogger.Println(color.GreenString("Subtitle file already exists for %s, skipping...", filePath))
		return
	}

	// Execute the ffmpeg command
	cmd := exec.Command("ffmpeg", "-i", filePath, "-map", "0:s:0", outputFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		errorLogger.Printf(color.RedString("Failed to execute ffmpeg command on %s: %v", filePath, err))
		return
	}

	// Success message
	successLogger.Println(color.YellowString("Successfully processed %s", filePath))
}
