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
        "path"
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

func (m *MarkdownFileRecord) PrependWrapper(content string) {
        err := m.Prepend(content)
        if err != nil {
                fmt.Println("[ERROR] There was an error writing hugo headers: ", err)
        }
}

/* ============================== */
/* End of modified record.go code */
/* ============================== */

// General function for regex
func regex_line_of_markdown(contents []string, regex string, variable string, line int) (value []string, line_number int) {
        if Index(contents, variable) >= 0 {
                re := regex.MustCompile(regex)
                value := re.FindAllString(contents[line], -1)
                // if we find it, move down two lines since every line in between new paragraphs is blank in markdown
                line_number := line + 2
                // delete the line where information was copied
                contents[line] = ""
                return
        }
        // didn't find anything, then leave blank and do not iterate the line number
        value = ""
        line_number = line
        return
}

// Read markdown document and write the hugo headers to the beginning of the document
func read_markdown_write_hugo_headers(md_file_path string, docx_file_path string, hugo_dir string, wg *sync.WaitGroup) {
        markdownfile := NewMarkdownFile(md_file_path)
        err := markdownfile.readMarkdownLines()
        if err != nil {
                fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
        }
        // Find the substrings for driveraker tags/categories, titles, subtitles, image captions, in-article headers, and bylines below:
        // REWRITE ALL THESE CHECKS TO BE MORE MODULAR (i.e. write another general function)
        var i int
        var tags []string
        i = 0
        tags, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_TAGS)](\w+)`, "DRVRKR\\_TAGS", i)
        // Now find the DRVRKR\_CATEGORIES
        var categories []string
        categories, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_CATEGORIES)](\w+)`, "DRVRKR\\_CATEGORIES", i)
        // Now find the DRVRKR\_PUB\_DATE
        var publicationyearmonthdate []string
        publicationyearmonthdate, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_PUB\\\_DATE)](\w+)`, "DRVRKR\\_PUB\\_DATE", i)
        // Now find the DRVRKR\_UPDATE\_DATE
        var updateyearmonthdate []string
        updateyearmonthdate, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_UPDATE\\\_DATE)](\w+)`, "DRVRKR\\_UPDATE\\_DATE", i)
        // Now find the cover photo for the article
        var imagenames []string
        var imagename string
        imagenames, i = regex_line_of_markdown(markdownfile.Contents, `(\w+.png)`, `<img src=`, i)
        imagename := imagenames[1]
        cover_image_path_before := path.Dir(path.Dir(docx_file_path)) + "/" + imagename
        cover_image_path_after := hugo_dir + "/static/images/" + imagename
        copy_cover_image := exec.Command("cp", cover_image_path_before + " " + cover_image_path_after)
        copy_cover_image.Dir = cover_image_path_before
        fmt.Println("Moving inline image to hugo directory...")
        out, err := copy_cover_image.Output()
        if err != nil {
                fmt.Println("[ERROR] Error moving" + imagename + ": ", err)
                return
        }
        // Now find the image caption
        var imagecaption []string
        imagecaption, i = regex_line_of_markdown(markdownfile.Contents, `##### +(.*)`, `#####`, i)
        // Now find the headline of the article
        var title []string
        title, i = regex_line_of_markdown(markdownfile.Contents, `# +(.*)`, `#`, i)
        // Find the subtitle
        var subtitle []string
        subtitle, i = regex_line_of_markdown(markdownfile.Contents, `# +(.*)`, `##`, i)
        // Find the authors on the byline
        var author_names []string
        author_names, i = regex_line_of_markdown(markdownfile.contents, `[^(####By |,and|,)](?:By | and)*?(\w+.\w+)`, `#### By`, i)
        // For-loop through the rest of the document looking for in-line images
        // in-line headers are taken care of on frontend by hugo's theme
        // in-line captions are taken care of on frontend by hugo's theme
        for j := i; j < len(markdownfile.Contents); j++; {
                if Index(markdownfile.Contents[j], `<img src=`) >= 0 {
                        re2 := regex.MustCompile(`(\w+.png)`)
                        inline_image := re2.FindAllStrings(markdownfile.Contents[j], -1)
                        inline_image_path_before := path.Dir(path.Dir(docx_file_path)) + "/" + inline_image[1]
                        inline_image_path_after := hugo_dir + "/static/images/" + inline_image[1]
                        copy_image := exec.Command("cp", inline_image_path_before + " " + inline_image_path_after)
                        copy_image.Dir = inline_image_path_before
                        fmt.Println("Moving inline image to hugo directory...")
                        out, err := copy_image.Output()
                        if err != nil {
                                fmt.Println("[ERROR] Error moving" + inline_image[1] + ": ", err)
                                return
                        }
                        fmt.Println("Done moving " + inline_image[1])
                        fmt.Println("Writing a new inline-image path for " + md_file_path)
                        // Use the image caption as the alt text for the inline-image
                        regex_alt_text := regex.MustCompile(`##### +(.*)`)
                        alt_text := regex_alt_text.FindAllString(markdownfile.Contents[j + 2], -1)
                        // Rewrite the inline image to have a css class called inline-image
                        markdownfile.Contents[j] = "<img src= \"" + inline_image_path_after + "\" alt=\"" + alt_text + "\" class=\"inline-image\">"
                }
        }
        // Now prepend the hugo JSON front-matter to the file
        // they will need to be prepended backwards
        markdownfile.PrependWrapper("}")
        // Add authors to hugo front-matter
        author_list := fmt.Sprintf("%f", author_names)
        author_list = strings.Replace(author_list, `%!f(string=`, `"`, -1)
        author_list = strings.Replace(author_list, `) `, `", `, -1)
        author_list = strings.Replace(author_list, `)`, `"`, -1)
        markdownfile.PrependWrapper("    \"authors\": " + author_list)
        // Add tags to hugo front-matter
        tag_list := fmt.Sprintf("%f", tags)
        tag_list = strings.Replace(tag_list, `%!f(string= `, `"`, -1)
        tag_list = strings.Replace(tag_list, `) `, `", `, -1)
        tag_list = strings.Replace(tag_list, `)`, `"`, -1)
        markdownfile.PrependWrapper("    \"tags\": " + tag_list)
        // Add categories to hugo front-matter
        cat_list := fmt.Sprintf("%f", categories)
        cat_list = strings.Replace(cat_list, `%!f(string= `, `"`, -1)
        cat_list = strings.Replace(cat_list, `) `, `", `, -1)
        cat_list = strings.Replace(cat_list, `)`, `"`, -1)
        markdownfile.PrependWrapper("    \"categories\": " + cat_list)
        // Mark article as not a draft in hugo front-matter
        markdownfile.PrependWrapper("    \"draft\": \"false\"")
        // Add image path to hugo front-matter
        markdownfile.PrependWrapper("    \"image\": \"" + imagename + "\"")
        // Add a last modified date to the hugo front-matter
        mod_date := fmt.Sprintf("%f", updateyearmonthdate)
        mod_date = strings.Replace(mod_date, `%!f(string= `, ``, -1)
        mod_date = strings.Replace(mod_date, `)`, ``, -1)
        mod_date = strings.Replace(mod_date, `[`, `"`, -1)
        mod_date = strings.Replace(mod_date, `]`, `"`, -1)
        mod_date = strings.Replace(mod_date, ` `, `-`, -1)
        markdownfile.PrependWrapper("    \"lastmod\": " + mod_date)
        // Add publication date to hugo front-matter
        // And only publish the article on the date or after it
        pub_date := fmt.Sprintf("%f", publicationyearmonthdate)
        pub_date = strings.Replace(pub_date, `%!f(string= `, ``, -1)
        pub_date = strings.Replace(pub_date, `)`, ``, -1)
        pub_date = strings.Replace(pub_date, `[`, `"`, -1)
        pub_date = strings.Replace(pub_date, `]`, `"`, -1)
        pub_date = strings.Replace(pub_date, ` `, `-`, -1)
        markdownfile.PrependWrapper("    \"publishDate\": " + pub_date)
        markdownfile.PrependWrapper("    \"date\": " + pub_date)
        // Add the subtitle as a description of the story to the hugo front-matter
        // Front end can use the subtitle as a brief description of the story for the front page
        // Front end style can make the description field a subtitle for the article page
        description := fmt.Sprintf("%f", subtitle)
        description = strings.Replace(description, `%!f(string= `, ``, -1)
        description = strings.Replace(description, `)`, ``, -1)
        description = strings.Replace(description, `[`, `"`, -1)
        description = strings.Replace(description, `]`, `"`, -1)
        markdownfile.PrependWrapper("    \"description\": " + description)
        // Add Title to the hugo front-matter
        headline := fmt.Sprintf("%f", title)
        headline = strings.Replace(headline, `%!f(string= `, ``, -1)
        headline = strings.Replace(headline, `)`, ``, -1)
        headline = strings.Replace(headline, `[`, `"`, -1)
        headline = strings.Replace(headline, `]`, `"`, -1)
        markdownfile.PrependWrapper("    \"title\": " + headline)
        // End the hugo JSON front-matter
        markdownfile.PrependWrapper("{")
}

// Use hugo to compile the markdown files into html and then serve with hugo or with nginx
func compile_and_serve_hugo_site(hugo_dir string, prod_dir string, use_hugo bool, wg *sync.WaitGroup) {
}
