package indexing

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"syscall"
	"time"
)

const (
	directoryWorkers       = 20
	fileWorkers            = 80
	directoryJobBufferSize = 100
	fileJobBufferSize      = 500
)

type collectedInfo struct {
	numOfFiles            int
	numOfDirectories      int
	fileNames             []string
	directories           []string
	numOfFilesWithContent int
	entryDetails          []*entryCollection
	numOfIgnoredEntries   int
	notRegistered         []*notAccessedPaths
	mu                    sync.Mutex
}

type notAccessedPaths struct {
	path string
	err  string
}

type entryCollection struct {
	fullPath    string // primary key
	parentDirID string // foreign key
	name        string
	isDir       bool
	size        int64
	//creationTime       int64 // Btim (not included syscall.Stat_t)
	modificationTime     int64  // os.fileStat.modTime or os.fileStat.sys.Mtim.Sec + Mtim.Nsec
	accessTime           int64  // os.fileStat.sys.Atim.Sec + Atim.Nsec
	metaDataChangeTime   int64  // os.fileStat.sys.Ctim.Sec + Ctim.Nsec
	ownerID              uint32 // os.fileStat.sys.Uid
	groupID              uint32 // os.fileStat.sys.Gid
	extension            string
	fileType             string // MIME type. Determined by file extension and/or internal magic bytes
	contentSnippet       []byte // short extract of the files content. [:500] to start with
	fullTextIndex        []byte // the complete textual content of a document, stored in separate Full-Text Search index
	lineCountTotal       int
	lineCountWithContent int
	//tags               []string // user defined tags or keywords from internal metadata
}

type readJob struct {
	path string
}

func readDir(path string, theWorks *collectedInfo, isRoot bool) {
	entry := entryCollection{}

	dirStat, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	entry.fullPath = path
	if !isRoot {
		entry.parentDirID = filepath.Dir(path)
	}
	entry.name = filepath.Base(path)
	entry.isDir = true
	entry.size = dirStat.Size()

	statT := dirStat.Sys().(*syscall.Stat_t)
	entry.modificationTime = statT.Mtim.Sec + statT.Mtim.Nsec
	entry.accessTime = statT.Atim.Sec + statT.Atim.Nsec
	entry.metaDataChangeTime = statT.Ctim.Sec + statT.Ctim.Nsec

	entry.ownerID = statT.Uid
	entry.groupID = statT.Gid
	entry.extension = filepath.Ext(entry.name)
	entry.fileType = filepath.Ext(entry.name)

	theWorks.mu.Lock()
	theWorks.numOfDirectories += 1
	theWorks.directories = append(theWorks.directories, path)
	theWorks.entryDetails = append(theWorks.entryDetails, &entry)
	theWorks.mu.Unlock()
}

func readFile(filename string, theWorks *collectedInfo) {
	entry := entryCollection{}

	contentsRead := false

	contentFiles := []string{".txt", ".md", ".go", ".py"}
	if slices.Contains(contentFiles, filepath.Ext(filename)) {
		contents, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		contentsRead = true
		lineCountTotal := bytes.Count(contents, []byte("\n"))
		blankLines := bytes.Count(contents, []byte("\n\n"))
		lineCountWithContent := lineCountTotal - blankLines

		contents = bytes.ReplaceAll(contents, []byte("\n"), []byte(" "))
		contents = bytes.ReplaceAll(contents, []byte("\r"), []byte(" "))
		contents = bytes.ReplaceAll(contents, []byte("\t"), []byte(" "))

		regExCleanup := regexp.MustCompile(`[\p{C}\p{Zl}\p{Zp}]`)
		contents = regExCleanup.ReplaceAll(contents, []byte(" "))
		contents = regexp.MustCompile(`\s+`).ReplaceAll(contents, []byte(" "))
		if len(contents) < 500 {
			entry.contentSnippet = contents
		} else {
			entry.contentSnippet = contents[:500]
		}
		entry.fullTextIndex = contents
		entry.lineCountTotal = lineCountTotal
		entry.lineCountWithContent = lineCountWithContent
	}

	fileStat, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}

	entry.fullPath = filename
	entry.parentDirID = filepath.Dir(filename)
	entry.name = filepath.Base(filename)
	entry.isDir = false
	entry.size = fileStat.Size()

	statT := fileStat.Sys().(*syscall.Stat_t)
	entry.modificationTime = statT.Mtim.Sec + statT.Mtim.Nsec
	entry.accessTime = statT.Atim.Sec + statT.Atim.Nsec
	entry.metaDataChangeTime = statT.Ctim.Sec + statT.Ctim.Nsec

	entry.ownerID = statT.Uid
	entry.groupID = statT.Gid

	theWorks.mu.Lock()
	theWorks.numOfFiles += 1
	if contentsRead {
		theWorks.numOfFilesWithContent += 1
	}
	theWorks.fileNames = append(theWorks.fileNames, filename)
	theWorks.entryDetails = append(theWorks.entryDetails, &entry)
	theWorks.mu.Unlock()
}

func traverseDirectory(root string, dirJobs chan<- readJob, fileJobs chan<- readJob, wg *sync.WaitGroup, theWorks *collectedInfo) {
	defer wg.Done()

	defer close(dirJobs)
	defer close(fileJobs)

	excludedEntries := []string{
		".cache",
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			failedPath := notAccessedPaths{path: path, err: err.Error()}
			theWorks.mu.Lock()
			theWorks.numOfIgnoredEntries += 1
			theWorks.notRegistered = append(theWorks.notRegistered, &failedPath)
			theWorks.mu.Unlock()
			return nil
		}

		if path == root {
			return nil
		}

		_, err = os.Stat(path)
		if err != nil {
			return nil
		}

		if d.IsDir() && slices.Contains(excludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		job := readJob{path: path}
		if d.IsDir() {
			dirJobs <- job
		} else {
			fileJobs <- job
		}

		return nil
	})

	if err != nil {
		log.Printf("Fatal error during directory traversal: %v", err)
	}
}

func dirWorker(readJobs <-chan readJob, wg *sync.WaitGroup, theWorks *collectedInfo) {
	defer wg.Done()

	for t := range readJobs {
		readDir(t.path, theWorks, false)
	}
}

func fileWorker(readJobs <-chan readJob, wg *sync.WaitGroup, theWorks *collectedInfo) {
	defer wg.Done()
	for t := range readJobs {
		readFile(t.path, theWorks)
	}
}

func Main() {
	start := time.Now()
	theWorks := collectedInfo{}

	fileReadJobs := make(chan readJob, fileJobBufferSize)
	dirReadJobs := make(chan readJob, directoryJobBufferSize)

	var wg sync.WaitGroup
	totalWorkers := 1 + directoryWorkers + fileWorkers
	wg.Add(totalWorkers)

	path := "/home/utled"
	stat, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	if !stat.IsDir() {
		log.Fatal("Starting path must be a directory")
	}
	readDir(path, &theWorks, true)

	for i := 0; i < directoryWorkers; i += 1 {
		go dirWorker(dirReadJobs, &wg, &theWorks)
	}

	for i := 0; i < fileWorkers; i += 1 {
		go fileWorker(fileReadJobs, &wg, &theWorks)
	}

	go traverseDirectory(path, dirReadJobs, fileReadJobs, &wg, &theWorks)

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println("Elapsed: ", elapsed)

	fmt.Println("Number of directories: ", theWorks.numOfDirectories)
	fmt.Println("Number of files: ", theWorks.numOfFiles)
	fmt.Println("Number read contents: ", theWorks.numOfFilesWithContent)
	fmt.Println("Number ignored entries: ", theWorks.numOfIgnoredEntries)
}
