package fileio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// BufferedWriter provides buffered file writing with automatic flushing
type BufferedWriter struct {
	file     *os.File
	writer   *bufio.Writer
	mu       sync.Mutex
	autoFlush bool
	flushInterval time.Duration
	stopFlush chan struct{}
}

// NewBufferedWriter creates a new buffered writer
func NewBufferedWriter(path string, bufferSize int, autoFlush bool) (*BufferedWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	bw := &BufferedWriter{
		file:     file,
		writer:   bufio.NewWriterSize(file, bufferSize),
		autoFlush: autoFlush,
		flushInterval: 5 * time.Second,
		stopFlush: make(chan struct{}),
	}

	if autoFlush {
		go bw.autoFlushLoop()
	}

	return bw, nil
}

// Write writes data to the buffer
func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.writer.Write(p)
}

// WriteString writes a string to the buffer
func (bw *BufferedWriter) WriteString(s string) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.writer.WriteString(s)
}

// Flush flushes the buffer to disk
func (bw *BufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.writer.Flush()
}

// Close closes the writer and file
func (bw *BufferedWriter) Close() error {
	if bw.autoFlush {
		close(bw.stopFlush)
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	if err := bw.writer.Flush(); err != nil {
		return err
	}

	return bw.file.Close()
}

// autoFlushLoop periodically flushes the buffer
func (bw *BufferedWriter) autoFlushLoop() {
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bw.Flush()
		case <-bw.stopFlush:
			return
		}
	}
}

// BatchFileReader provides efficient batch reading of files
type BatchFileReader struct {
	files    []string
	current  int
	reader   *bufio.Reader
	file     *os.File
	mu       sync.Mutex
}

// NewBatchFileReader creates a new batch file reader
func NewBatchFileReader(files []string) *BatchFileReader {
	return &BatchFileReader{
		files:   files,
		current: -1,
	}
}

// NextFile opens the next file in the batch
func (br *BatchFileReader) NextFile() (string, error) {
	br.mu.Lock()
	defer br.mu.Unlock()

	// Close current file if open
	if br.file != nil {
		br.file.Close()
		br.file = nil
		br.reader = nil
	}

	br.current++
	if br.current >= len(br.files) {
		return "", io.EOF
	}

	filename := br.files[br.current]
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	br.file = file
	br.reader = bufio.NewReader(file)

	return filename, nil
}

// Read reads from the current file
func (br *BatchFileReader) Read(p []byte) (n int, err error) {
	br.mu.Lock()
	defer br.mu.Unlock()

	if br.reader == nil {
		return 0, fmt.Errorf("no file open")
	}

	return br.reader.Read(p)
}

// ReadLine reads a line from the current file
func (br *BatchFileReader) ReadLine() (string, error) {
	br.mu.Lock()
	defer br.mu.Unlock()

	if br.reader == nil {
		return "", fmt.Errorf("no file open")
	}

	line, err := br.reader.ReadString('\n')
	return line, err
}

// Close closes the reader
func (br *BatchFileReader) Close() error {
	br.mu.Lock()
	defer br.mu.Unlock()

	if br.file != nil {
		return br.file.Close()
	}

	return nil
}

// FileCache provides a simple file content cache
type FileCache struct {
	cache map[string]cachedFile
	mu    sync.RWMutex
	ttl   time.Duration
}

type cachedFile struct {
	content   []byte
	modTime   time.Time
	cacheTime time.Time
}

// NewFileCache creates a new file cache
func NewFileCache(ttl time.Duration) *FileCache {
	return &FileCache{
		cache: make(map[string]cachedFile),
		ttl:   ttl,
	}
}

// Read reads a file, using cache if available and valid
func (fc *FileCache) Read(path string) ([]byte, error) {
	fc.mu.RLock()
	cached, exists := fc.cache[path]
	fc.mu.RUnlock()

	if exists {
		// Check if cache is still valid
		info, err := os.Stat(path)
		if err == nil {
			if info.ModTime().Equal(cached.modTime) &&
			   time.Since(cached.cacheTime) < fc.ttl {
				return cached.content, nil
			}
		}
	}

	// Read from disk
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return content, nil // Return content but don't cache
	}

	// Update cache
	fc.mu.Lock()
	fc.cache[path] = cachedFile{
		content:   content,
		modTime:   info.ModTime(),
		cacheTime: time.Now(),
	}
	fc.mu.Unlock()

	return content, nil
}

// Invalidate removes a file from cache
func (fc *FileCache) Invalidate(path string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	delete(fc.cache, path)
}

// Clear clears the entire cache
func (fc *FileCache) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache = make(map[string]cachedFile)
}

// AtomicWriter provides atomic file writing
type AtomicWriter struct {
	targetPath string
	tempFile   *os.File
}

// NewAtomicWriter creates a new atomic writer
func NewAtomicWriter(path string) (*AtomicWriter, error) {
	tempFile, err := os.CreateTemp("", "skm-atomic-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	return &AtomicWriter{
		targetPath: path,
		tempFile:   tempFile,
	}, nil
}

// Write writes data to the temporary file
func (aw *AtomicWriter) Write(p []byte) (n int, err error) {
	return aw.tempFile.Write(p)
}

// Commit commits the write by renaming temp file to target
func (aw *AtomicWriter) Commit() error {
	if err := aw.tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := aw.tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(aw.tempFile.Name(), aw.targetPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Abort aborts the write and removes temp file
func (aw *AtomicWriter) Abort() error {
	aw.tempFile.Close()
	return os.Remove(aw.tempFile.Name())
}

