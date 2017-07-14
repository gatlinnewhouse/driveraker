package main

import (
        "bufio"
        "crypto/md5"
        "encoding/json"
        "encoding/hex"
        "errors"
        "fmt"
        "io"
        "log"
        "os"
        "os/exec"
        "regexp"
        "sync"
        "time"
)


/* ========================= */
/* The follow code is forked */
/* from github.com/nf/goto   */
/* ========================= */
const (
        saveTimeout     = 10e9
        saveQueueLength = 1000
)

type Store interface {
        Put(path, key *string) error
        Get(key, path *string) error
}

type PathStore struct {
        mu sync.RWMutex
        paths map[string]string
        count int
        save chan record
}

type record struct {
        key, path string
}

// Use md5 hash sums for the filepaths
func md5hash(text string, DriveSyncDirectory string) string {
        r := strings.NewReplacer(DriveSyncDirectory, "")
        relative-path := r.Replace(text)
        hasher := md5.New()
        hasher.Write([]byte(relative-path))
        return hex.EncodeToString(hasher.Sum(nil))
}

// Create a hashtable of paths and keys
func NewPathStore(filename string) *PathStore {
        s := &PathStore{paths: make(map[string]string)}
        if filename != "" {
                s.save = make(chan record, saveQueueLength)
                if err := s.load(filename); err != nil {
                        log.Println("[ERROR] Error storing paths: ", err)
                }
                go s.saveLoop(filename)
        }
        return s
}

// Check for a path in the hashtable
func (s *PathStore) Get(key, path *string) bool {
        s.mu.RLock()
        defer s.mu.RUnlock()
        if p, okay := s.paths[*key]; ok {
                *path = u
                return nil
        }
        return errors.New("Key not found")
}

// Write a new path to the hashtable for an known key
func (s *PathStore) Set(key, path *string) bool {
        s.mu.Lock()
        defer s.mu.Unlock()
        if _, present := s.paths[*key]; present {
                return errors.New("Key already exists")
        }
        // Otherwise add the new path
        s.paths[*key] = *path
        return nil
}

// Write a new path to the hashtable without a known key
func (s *PathStore) Put(path, DriveSyncDirectory, key *string) error {
        for {
                *key = md5hash(path, DriveSyncDirectory)
                s.count++
                if err := s.Set(key, path); err == nil {
                        break
                }
        }
        if s.save != nil {
                s.save <- record{*key, *path}
        }
        return nil
}

// Load the hashtable from a file
func (s *PathStore) load(filename string) error {
        f, err := os.Open(filename)
        if err != nil {
                return err
        }
        defer f.Close()
        b := bufio.NewReader(f)
        d := json.NewDecoder(b)
        for {
                var r record
                if err := d.Decode(&r); err == io.EOF {
                        break
                } else if err != nil {
                        return err
                }
                if err = s.Set(&r.key, &r.path); err != nil {
                        return err
                }
        }
        return nil
}

// Save the hashtable to a file
func (s *PathStore) saveLoop(filename string) {
        f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
        if err != nil {
                log.Println("PathStore: ", err)
                return
        }
        b := bufio.NewWriter(f)
        e := json.NewEncoder(b)
        t := time.NewTicker(saveTimeout)
        defer f.Close()
        defer b.Flush()
        for {
                var err error
                select {
                case r:= <-s.save:
                        err = e.Encode(r)
                case <-t.C:
                        err = b.Flush()
                }
                if err != nil {
                        log.Println("PathStore: ", err)
                }
        }
}
/* ============================= */
/* End of modified /nf/goto code */
/* ============================= */


// The configuration file struct
type Configuration struct {
        DriveSyncDirectory string
        GoogleDriveRemoteDirectory string
        HugoPostDirectory string
}

// Read the configuration JSON file in order to get some settings and directories
func read_cfg(filename string, wg *sync.WaitGroup, conf_message chan string) {
        file, _ := os.Open(filename)
        decoder := json.NewDecoder(file)
        configuration := Configuration{}
        err := decoder.Decode(&configuration)
        if err != nil {
                fmt.Println("[ERROR] Error reading the JSON confguration: ", err)
                return
        }
        drive_sync_dir := fmt.Sprintf(configuration.DriveSyncDirectory)
        conf_message <- drive_sync_dir
        drive_remote_dir := fmt.Sprintf(configuration.GoogleDriveRemoteDirectory)
        conf_message <- drive_remote_dir
        hugo_post_dir := fmt.Sprintf(configuration.HugoPostDirectory)
        conf_message <- hugo_post_dir
        wg.Done()
}

// Sync google drive remote folder to the configured local directory.
// Then send the output from drive CLI to a function to intepret the output
// by stripping the full output down to an array of string paths to docx files.
func sync_google_drive(sync_dir string, drive_remote_dir string, wg *sync.WaitGroup) {
        sync_gd := new(sync.WaitGroup)
        output := make(chan string)
        sync := exec.Command("drive pull -no-prompt -desktop-links=false -export docx", drive_remote_dir)
        sync.Dir = sync_dir
        fmt.Println("Syncing Google Drive...")
        out, err := sync.Output()
        if err != nil {
                fmt.Println("[ERROR] Error syncing Google Drive: ", err)
                return
        }
        fmt.Println("Done syncing!")
        sync_gd.Add(1)
        go intepret_drive_output(sync_gd, output)
        output <- string(out)
        sync_gd.Wait()
        wg.Done()
}

// Find all Exported file paths via a regex expression and then add them to an array
func interpret_drive_output(sync_gd *sync.WaitGroup, output chan string) {
        results := <-output
        re := regexp.MustCompile(`to '(.*?)'`)
        matches := re.FindAllString(results, -1)
        sync_gd.Done()
}

// Convert from docx to markdown with pandoc
func convert_to_markdown_with_pandoc(docx_file_path string, md_file_path string, wg *sync.WaitGroup) {
        convert := exec.Command("pandoc --atx-headers --smart --normalize --email-obfuscation=references --mathjax -t markdown_strict -o", md_file_path, docx_file_path)
        out, err := convert.Output()
        if err != nil {
                fmt.Println("[ERROR] Error converting files to markdown with pandoc: ", err)
        }
        wg.Done()
}


/* =================================================================== */
/* The following code forked from:                                     */
/* https://gist.github.com/toruuetani/f6aa4751a66ef65646c1a4934471396b */
/* =================================================================== */
type MarkdownFileRecord struct {
        Filename string
        Contents []string
}

func NewMarkdownFile(filename string) *MarkdownFileRecord {
        return &MarkdownFileRecord {
                Filename: filename,
                Contents: make([]string, 0),
        }
}

func (m *MarkdownFileRecord) readMarkdownLines() error {
        if _, err := os.Stat(m.Filename); err != nil {
                return nil
        }
        f, err := os.OpenFile(m.Filename, os.O_RDONLY, 0600)
        if err != nil {
                return err
        }
        defer f.Close()

        scanner := bufio.NewScanner(f)
        for scanner.Scan() {
                if tmp := scanner.Text(); len(tmp) != 0 {
                        m.Contents = append(m.Contents, tmp)
                }
        }
}

func (m *MarkdownFileRecord) Prepend(content string) error {
        err := m.readLines()
        if err != nil {
                return err
        }

        f, err := os.OpenFile(m.Filename, os.O_CREATE|os.O_WRONLY, 0600)
        if err != nil {
                return err
        }
        defer f.Close()

        writer := bufio.NewWriter(f)
        writer.WriteString(fmt.Sprintf("%s\n", content))
        for _, line := range m.Contents {
                _, err := writer.WriteString(fmt.Sprintf("%s\n", line))
                if err != nil {
                        return err
                }
        }

        if err := writer.Flush(); err != nil {
                return err
        }

        return nil
}
/* ============================== */
/* End of modified record.go code */
/* ============================== */

// Add the hugo headers to the markdown file
func add_hugo_headers(md_file_path string, wg *sync.WaitGroup) {
}

// Use hugo to compile the markdown files into html and then serve with hugo or with nginx
func compile_and_serve_hugo_site(hugo_dir string, prod_dir string, use_hugo bool, wg *sync.WaitGroup) {
}
