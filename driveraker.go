package main

import (
        "encoding/json"
        "fmt"
        "os"
        "os/exec"
        "log"
        "sync"
)

type Configuration struct {
        DriveSyncDirectory string
        GoogleDriveRemoteDirectory string
        HugoPostDirectory string
}

func exe_cmd(cmd string, arg1 string, wg *sync.WaitGroup) {
        out, err := exec.Command(cmd, arg1).Output()
        if err != nil {
                log.Fatal(err)
        }
        fmt.Printf("The output is %s", out)
        wg.Done()
}

func read_cfg(filename string, wg *sync.WaitGroup, conf_message chan string) {
        file, _ := os.Open(filename)
        decoder := json.NewDecoder(file)
        configuration := Configuration{}
        err := decoder.Decode(&configuration)
        if err != nil {
                fmt.Println("error: ", err)
        }
        drive_sync_dir := fmt.Sprintf(configuration.DriveSyncDirectory)
        conf_message <- drive_sync_dir
        drive_remote_dir := fmt.Sprintf(configuration.GoogleDriveRemoteDirectory)
        conf_message <- drive_remote_dir
        hugo_post_dir := fmt.Sprintf(configuration.HugoPostDirectory)
        conf_message <- hugo_post_dir
        wg.Done()
}

func sync_google_drive(sync_dir string, drive_remote_dir string, wg *sync.WaitGroup) {
        sync := exec.Command("drive pull -desktop-links=false -export docx", drive_remote_dir)
        sync.Dir = sync_dir
        fmt.Println("Syncing Google Drive...")
        out, err := sync.Output()
        if err != nil {
                fmt.Println("[ERROR] Error syncing Google Drive: ", err)
        }
        fmt.Println("Done syncing!")
        wg.Done()
}

func convert_to_markdown_with_pandoc(docx_file_path string, md_file_path string, wg *sync.WaitGroup) {
        convert := exec.Command("pandoc --atx-headers --smart --normalize --email-obfuscation=references --mathjax -t markdown_strict -o", md_file_path, docx_file_path)
        out, err := convert.Output()
        if err != nil {
                fmt.Println("[ERROR] Error converting files to markdown with pandoc: ", err)
        }
        wg.Done()
}

func compile_and_serve_hugo_site(hugo_dir string, prod_dir string, use_hugo bool, wg *sync.WaitGroup) {
}

func main() {
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
}
