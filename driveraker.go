package main

import (
        "crypt/md5"
        "encoding/json"
        "encoding/hex"
        "fmt"
        "os"
        "os/exec"
        "log"
        "regexp"
        "sync"
)

//The mutex lock for reading/writing to the hashtable
var counter = struct {
        sync.RWMutex
        hashtable map[string]string
}{hashtable: make(map[string]string)}

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

// Use md5 hash sums for the filepaths
func md5hash(text string, DriveSyncDirectory string) string {
        r := strings.NewReplacer(DriveSyncDirectory, "")
        relative-path := r.Replace(text)
        hasher := md5.New()
        hasher.Write([]byte(relative-path))
        return hex.EncodeToString(hasher.Sum(nil))
}

// Write the filepath string to the hashtable
func write_path_to_hashtable(path string, hashtable map[string]string) {
        key := md5hash(path)
        counter.Lock()
        counter.m[key]path
        counter.Unlock()
}

// Lookup a path in the hashtable, if it exists return true, otherwise false
func check_for_path(path string, hashtable map[string]string, read *sync.WaitGroup) bool {
        key := md5hash(path)
        counter.RLock()
        read_path := counter.hashtable[key]
        if (read_path == path) {
                return true
        }
        return false
}

// Write the hashtable to a json file in order to keep a backup in case of system reboot
func write_hashtable_to_json(hashtable map[string]string) {
        bytes, err := json.Marshal(hashtable)
        if err != nil {
                fmt.Println("[ERROR] Error writing hashtable to JSON: ", err)
                return
        }
        text := string(bytes)
        fmt.Println(bytes)
}

// Read the saved hashtable from the saved json file to restart the syncing after reboot
func read_hashtable_from_json(hashtable map[string]string, table_dir string) {
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

// Use hugo to compile the markdown files into html and then serve with hugo or with nginx
func compile_and_serve_hugo_site(hugo_dir string, prod_dir string, use_hugo bool, wg *sync.WaitGroup) {
}

/* func main() {
        conf_message := make(chan string)
        wg := new(sync.WaitGroup)
        wg.Add(2)
        go exe_cmd("echo", "ping", wg)
        go read_cfg("conf.json", wg, conf_message)
        drive_sync_dir := <-conf_message
        fmt.Println(drive_sync_dir)
        drive_remote_dir := <-conf_message
        fmt.Println(drive_remote_dir)
        hugo_post_dir := <-conf_message
        fmt.Println(hugo_post_dir)
        wg.Wait()
} */
