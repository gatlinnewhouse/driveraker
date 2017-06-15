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
        hugo_post_dir := fmt.Sprintf(configuration.HugoPostDirectory)
        conf_message <- hugo_post_dir
        wg.Done()
}

func main() {
        conf_message := make(chan string)
        wg := new(sync.WaitGroup)
        wg.Add(2)
        go exe_cmd("echo", "ping", wg)
        go read_cfg("conf.json", wg, conf_message)
        drive_sync_dir := <- conf_message
        fmt.Println(drive_sync_dir)
        hugo_post_dir := <- conf_message
        fmt.Println(hugo_post_dir)
        wg.Wait()
}
