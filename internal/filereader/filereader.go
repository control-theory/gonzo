package filereader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileReader manages reading from multiple files with optional follow mode
type FileReader struct {
	filePaths  []string
	follow     bool
	ctx        context.Context
	cancel     context.CancelFunc
	lineChan   chan string
	wg         sync.WaitGroup
	mu         sync.Mutex
	watchers   map[string]*fsnotify.Watcher // Track file watchers for follow mode
	fileStates map[string]*fileState        // Track file states for follow mode
}

// fileState tracks the current position and state of a file being followed
type fileState struct {
	file     *os.File
	scanner  *bufio.Scanner
	size     int64
	modified time.Time
}

// New creates a new FileReader with the given file paths and options
func New(filePaths []string, follow bool) (*FileReader, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no file paths provided")
	}

	// Expand glob patterns
	expandedPaths, err := expandGlobs(filePaths)
	if err != nil {
		return nil, fmt.Errorf("error expanding file globs: %w", err)
	}

	if len(expandedPaths) == 0 {
		return nil, fmt.Errorf("no files found matching the provided patterns")
	}

	// Verify files exist and are readable
	var validPaths []string
	for _, path := range expandedPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			validPaths = append(validPaths, path)
		} else {
			log.Printf("Warning: skipping file %s: %v", path, err)
		}
	}

	if len(validPaths) == 0 {
		return nil, fmt.Errorf("no valid readable files found")
	}

	ctx, cancel := context.WithCancel(context.Background())

	fr := &FileReader{
		filePaths:  validPaths,
		follow:     follow,
		ctx:        ctx,
		cancel:     cancel,
		lineChan:   make(chan string, 100),
		watchers:   make(map[string]*fsnotify.Watcher),
		fileStates: make(map[string]*fileState),
	}

	return fr, nil
}

// expandGlobs expands glob patterns and returns sorted list of matching files
func expandGlobs(patterns []string) ([]string, error) {
	var allPaths []string
	pathSet := make(map[string]bool) // Use set to avoid duplicates

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
		}

		for _, match := range matches {
			// Get absolute path for consistency
			absPath, err := filepath.Abs(match)
			if err == nil {
				pathSet[absPath] = true
			}
		}
	}

	// Convert set to sorted slice
	for path := range pathSet {
		allPaths = append(allPaths, path)
	}
	sort.Strings(allPaths)

	return allPaths, nil
}

// Start begins reading from the files
func (fr *FileReader) Start() <-chan string {
	if fr.follow {
		fr.startFollowMode()
	} else {
		fr.startReadMode()
	}
	return fr.lineChan
}

// startReadMode reads files once from beginning to end
func (fr *FileReader) startReadMode() {
	fr.wg.Go(func() {
		defer close(fr.lineChan)

		for _, filePath := range fr.filePaths {
			if err := fr.readFile(filePath); err != nil {
				log.Printf("Error reading file %s: %v", filePath, err)
				continue
			}
		}
	})
}

// startFollowMode reads files and then watches for new content
func (fr *FileReader) startFollowMode() {
	fr.wg.Go(func() {
		defer close(fr.lineChan)
		defer fr.closeAllWatchers()

		// First, read existing content of all files
		for _, filePath := range fr.filePaths {
			if err := fr.readFile(filePath); err != nil {
				log.Printf("Error reading file %s: %v", filePath, err)
				continue
			}
		}

		// Then set up watchers for follow mode
		for _, filePath := range fr.filePaths {
			if err := fr.setupFileWatcher(filePath); err != nil {
				log.Printf("Error setting up watcher for %s: %v", filePath, err)
			}
		}

		// Keep running until context is cancelled
		<-fr.ctx.Done()
	})
}

// readFile reads a file from beginning to end
func (fr *FileReader) readFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Set larger buffer size for long log lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-fr.ctx.Done():
			return nil
		case fr.lineChan <- scanner.Text():
		}
	}

	return scanner.Err()
}

// setupFileWatcher sets up a file system watcher for follow mode
func (fr *FileReader) setupFileWatcher(filePath string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Get initial file info
	info, err := os.Stat(filePath)
	if err != nil {
		watcher.Close()
		return err
	}

	// Open file and position at end
	file, err := os.Open(filePath)
	if err != nil {
		watcher.Close()
		return err
	}

	// Seek to end of file
	currentSize, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		watcher.Close()
		return err
	}

	scanner := bufio.NewScanner(file)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	// Store file state
	fr.mu.Lock()
	fr.watchers[filePath] = watcher
	fr.fileStates[filePath] = &fileState{
		file:     file,
		scanner:  scanner,
		size:     currentSize,
		modified: info.ModTime(),
	}
	fr.mu.Unlock()

	// Add file to watcher
	if err := watcher.Add(filePath); err != nil {
		fr.cleanupFile(filePath)
		return err
	}

	// Start watching for changes
	fr.wg.Go(func() {
		fr.watchFile(filePath, watcher)
	})

	return nil
}

// watchFile watches a single file for changes
func (fr *FileReader) watchFile(filePath string, watcher *fsnotify.Watcher) {

	for {
		select {
		case <-fr.ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				fr.handleFileWrite(filePath)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error for %s: %v", filePath, err)
		}
	}
}

// handleFileWrite handles when a file is written to
func (fr *FileReader) handleFileWrite(filePath string) {
	fr.mu.Lock()
	state, exists := fr.fileStates[filePath]
	fr.mu.Unlock()

	if !exists {
		return
	}

	// Check if file was truncated (common with log rotation)
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}

	if info.Size() < state.size {
		// File was truncated, reopen from beginning
		fr.reopenFile(filePath)
		return
	}

	// Read new lines
	for state.scanner.Scan() {
		select {
		case <-fr.ctx.Done():
			return
		case fr.lineChan <- state.scanner.Text():
		}
	}

	// Update file state
	fr.mu.Lock()
	if state.file != nil {
		if pos, err := state.file.Seek(0, io.SeekCurrent); err == nil {
			state.size = pos
		}
		state.modified = info.ModTime()
	}
	fr.mu.Unlock()
}

// reopenFile reopens a file that may have been rotated
func (fr *FileReader) reopenFile(filePath string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	state, exists := fr.fileStates[filePath]
	if !exists {
		return
	}

	// Close old file
	if state.file != nil {
		state.file.Close()
	}

	// Open new file
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error reopening file %s: %v", filePath, err)
		return
	}

	// Create new scanner
	scanner := bufio.NewScanner(file)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	// Update state
	state.file = file
	state.scanner = scanner
	state.size = 0

	log.Printf("Reopened file %s (likely rotated)", filePath)
}

// cleanupFile closes and removes tracking for a file
func (fr *FileReader) cleanupFile(filePath string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	if state, exists := fr.fileStates[filePath]; exists {
		if state.file != nil {
			state.file.Close()
		}
		delete(fr.fileStates, filePath)
	}

	if watcher, exists := fr.watchers[filePath]; exists {
		watcher.Close()
		delete(fr.watchers, filePath)
	}
}

// closeAllWatchers closes all file watchers
func (fr *FileReader) closeAllWatchers() {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	for filePath := range fr.watchers {
		fr.cleanupFile(filePath)
	}
}

// Stop stops the file reader and closes all resources
func (fr *FileReader) Stop() {
	fr.cancel()
}

// Wait waits for all reading goroutines to finish
func (fr *FileReader) Wait() {
	fr.wg.Wait()
}

// GetFilePaths returns the list of files being read
func (fr *FileReader) GetFilePaths() []string {
	return append([]string{}, fr.filePaths...)
}