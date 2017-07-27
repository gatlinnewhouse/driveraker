package main

import (
	"bufio"
	"bytes"
	//	"crypto/md5"
	//	"encoding/hex"
	"encoding/json"
	//	"errors"
	"fmt"
	"io"
	"io/ioutil"
	//	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strings"
	"sync"
	//	"time"
)

/* ========================= */
/* The follow code is forked */
/* from github.com/nf/goto   */
/* ========================= */

/* not needed right now
const (
	saveTimeout     = 10e9
	saveQueueLength = 1000
)

type Store interface {
	Put(path, key *string) error
	Get(key, path *string) error
}

type PathStore struct {
	mu    sync.RWMutex
	paths map[string]string
	count int
	save  chan record
}

type record struct {
	key, path string
}

// Use md5 hash sums for the filepaths
func md5hash(text string, DriveSyncDirectory string) string {
	r := strings.NewReplacer(DriveSyncDirectory, "")
	relativepath := r.Replace(text)
	hasher := md5.New()
	hasher.Write([]byte(relativepath))
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
		case r := <-s.save:
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
	DriveSyncDirectory         string
	GoogleDriveRemoteDirectory string
	HugoPostDirectory          string
	NginxDirectory             string
}

// Read the configuration JSON file in order to get some settings and directories
func read_cfg(filename string, conf *sync.WaitGroup, conf_message chan string) {
	fmt.Println("Reading configuration...")
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
	nginx_dir := fmt.Sprintf(configuration.NginxDirectory)
	conf_message <- nginx_dir
	fmt.Println("Finished reading configuration!")
	conf.Done()
}

// Sync google drive remote folder to the configured local directory.
// Then send the output from drive CLI to a function to intepret the output
// by stripping the full output down to an array of string paths to docx files.
func sync_google_drive(sync_dir string, drive_remote_dir string, drive_sync *sync.WaitGroup, docx_paths_message chan []string) {
	sync_gd := new(sync.WaitGroup)
	output := make(chan string)
	file_paths := make(chan []string)
	sync := exec.Command("/usr/bin/drive", "pull", "-no-prompt", "-desktop-links=false", "-export", "docx", drive_remote_dir)
	sync.Dir = sync_dir
	fmt.Println("Syncing Google Drive...")
	out, err := sync.Output()
	if err != nil {
		fmt.Println("[ERROR] Error syncing Google Drive: ", err)
		return
	}
	fmt.Printf("drive: " + string(out))
	fmt.Println("Done syncing!")
	sync_gd.Add(1)
	go interpret_drive_output(sync_gd, output, file_paths)
	output <- string(out)
	docx_paths := <-file_paths
	sync_gd.Wait()
	docx_paths_message <- docx_paths
	drive_sync.Done()
}

// Find all Exported file paths via a regex expression and then add them to an array
func interpret_drive_output(sync_gd *sync.WaitGroup, output chan string, file_paths chan []string) {
	fmt.Println("Interpreting command line output...")
	results := <-output
	re := regexp.MustCompile(`[^'](?:to ')(.*?)'`)
	matches := re.FindAllString(results, -1)
	fmt.Printf("File paths: %s \n", matches)
	file_paths <- matches
	fmt.Println("Done!")
	sync_gd.Done()
}

// Convert from docx to markdown with pandoc
func convert_to_markdown_with_pandoc(docx_file_path string, md_file_path string, pandoc *sync.WaitGroup) {
	convert := exec.Command("/usr/bin/pandoc", "--atx-headers", "--smart", "--normalize", "--email-obfuscation=references", "--mathjax", "-t", "markdown_strict", "-o", md_file_path, docx_file_path)
	convert.Dir = "/"
	out, err := convert.CombinedOutput()
	if err != nil {
		fmt.Println("[ERROR] Error converting files to markdown with pandoc: ", err )
	}
	fmt.Println("pandoc: ", out)
	pandoc.Done()
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
	return &MarkdownFileRecord{
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
		tmp := scanner.Text()
		m.Contents = append(m.Contents, tmp)
	}
	f.Close()
	return nil
}

func (m *MarkdownFileRecord) Prepend(content []string) error {
	err := m.readMarkdownLines()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(m.Filename, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	for i := 0; i < len(content); i++ {
		writer.WriteString(fmt.Sprintf("%s\n", content[i]))
	}
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

func prependWrapper(content []string, md_file_path string, prepend *sync.WaitGroup) {
	err := NewMarkdownFile(md_file_path).Prepend(content)
	if err != nil {
		fmt.Println("[ERROR] Error prepending hugo front-matter to document: ", err)
	}
	prepend.Done()
}
/* ================================================================================================ */
/* End of modified record.go code.                                                                  */
/* Beginning of forked popline.go code from:                                                        */
/* https://stackoverflow.com/questions/30940190/remove-first-line-from-text-file-in-golang#30948278 */
/* ================================================================================================ */
func deleteLine(f *os.File) ([]byte, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, fi.Size()))
	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}
	line, err := buf.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	nw, err := io.Copy(f, buf)
	if err != nil {
		return nil, err
	}
	err = f.Truncate(nw)
	if err != nil {
		return nil, err
	}
	err = f.Sync()
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	return []byte(line), nil
}

func deleteLineWrapper(md_file_path string, deleteline *sync.WaitGroup) {
	f, err := os.OpenFile(md_file_path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("[ERROR] Error opening file: ", err)
	}
	defer f.Close()
	line, err := deleteLine(f)
	if err != nil {
		fmt.Println("[ERROR] Error deleting a line: ", err)
	}
	fmt.Printf("Deleted line: %s from %s\n", string(line), md_file_path)
	f.Close()
	deleteline.Done()
}
/* =============================== */
/* End of modified popline.go code */
/* =============================== */

// Rewrite a line in a file
func rewriteMarkdownLine(line int, replacement string, md_file_path string, rewritemarkdown *sync.WaitGroup) {
	input, err := ioutil.ReadFile(md_file_path)
	if err != nil {
		fmt.Println("[ERROR] Error opening the file", err)
	}
	contents := strings.Split(string(input), "\n")
	contents[line] = replacement
	output := strings.Join(contents, "\n")
	err = ioutil.WriteFile(md_file_path, []byte(output), 0644)
	if err != nil {
		fmt.Println("[ERROR] There was an error writing the file")
	}
	rewritemarkdown.Done()
}

// General function for regex
func regex_line_of_markdown(contents []string, regex string, variable string, line int) (value []string, line_number int) {
	if strings.Index(contents[line], variable) >= 0 {
		re := regexp.MustCompile(regex)
		value = re.FindAllString(contents[line], -1)
		// if we find it, move down a line
		line_number = line + 2
		return
	}
	value = append(value, "")
	line_number = line
	// didn't find anything, then leave blank and do not iterate the line number
	return
}

// Read markdown document and write the hugo headers to the beginning of the document
func read_markdown_write_hugo_headers(md_file_path string, docx_file_path string, hugo_dir string, nginx_dir string, front_matter *sync.WaitGroup) {
	markdownfile := NewMarkdownFile(md_file_path)
	err := markdownfile.readMarkdownLines()
	if err != nil {
		fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
	}
	// Read and then rewrite the line read according to what value it should be
	var i int // The number of driveraker front matter lines
	i = 0 // For the reading line, start at 0
	var hugoFrontMatter []string // Add all hugo front matter to this string slice
	hugoFrontMatter = append(hugoFrontMatter, "{")
	// Find DRVRKR\_TAGS
	var tags []string
	tags, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_TAGS)](\w+)`, "DRVRKR\\_TAGS", i)
	tag_list := fmt.Sprintf("%f", tags)
	tag_list = strings.Replace(tag_list, `%!f(string= `, `"`, -1)
	tag_list = strings.Replace(tag_list, `) `, `", `, -1)
	tag_list = strings.Replace(tag_list, `)`, `"`, -1)
	tag_list = "    \"tags\": " + tag_list
	hugoFrontMatter = append(hugoFrontMatter, tag_list)
	// Now find the DRVRKR\_CATEGORIES
	var categories []string
	categories, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_CATEGORIES)](\w+)`, "DRVRKR\\_CATEGORIES", i)
	cat_list := fmt.Sprintf("%f", categories)
	cat_list = strings.Replace(cat_list, `%!f(string= `, `"`, -1)
	cat_list = strings.Replace(cat_list, `) `, `", `, -1)
	cat_list = strings.Replace(cat_list, `)`, `"`, -1)
	cat_list = "    \"categories\": " + cat_list
	hugoFrontMatter = append(hugoFrontMatter, cat_list)
	// Draft status
	hugoFrontMatter = append(hugoFrontMatter, "    \"draft\": \"false\"")
	// Now find the DRVRKR\_PUB\_DATE
	var publicationyearmonthdate []string
	publicationyearmonthdate, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_PUB\\\_DATE)](\w+)`, "DRVRKR\\_PUB\\_DATE", i)
	pub_date := fmt.Sprintf("%f", publicationyearmonthdate)
	pub_date = strings.Replace(pub_date, `%!f(string= `, ``, -1)
	pub_date = strings.Replace(pub_date, `)`, ``, -1)
	pub_date = strings.Replace(pub_date, `[`, `"`, -1)
	pub_date = strings.Replace(pub_date, `]`, `"`, -1)
	pub_date = strings.Replace(pub_date, ` `, `-`, -1)
	hugoFrontMatter = append(hugoFrontMatter, "    \"date\": " + pub_date)
	hugoFrontMatter = append(hugoFrontMatter, "    \"publishDate\": " + pub_date)
	// Now find the DRVRKR\_UPDATE\_DATE
	var updateyearmonthdate []string
	updateyearmonthdate, i = regex_line_of_markdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_UPDATE\\\_DATE)](\w+)`, "DRVRKR\\_UPDATE\\_DATE", i)
	mod_date := fmt.Sprintf("%f", updateyearmonthdate)
	mod_date = strings.Replace(mod_date, `%!f(string= `, ``, -1)
	mod_date = strings.Replace(mod_date, `)`, ``, -1)
	mod_date = strings.Replace(mod_date, `[`, `"`, -1)
	mod_date = strings.Replace(mod_date, `]`, `"`, -1)
	mod_date = strings.Replace(mod_date, ` `, `-`, -1)
	mod_date = "    \"lastmod\": " + mod_date
	hugoFrontMatter = append(hugoFrontMatter, mod_date)
	// Now find the cover photo for the article
	var imagenames []string
	imagenames, i = regex_line_of_markdown(markdownfile.Contents, `(\w+.png)`, `<img src=`, i)
	imagename := imagenames[1]
	cover_image_path_before := path.Dir(path.Dir(docx_file_path)) + "/" + imagename
	//fmt.Println("image path before: " + "\"" + cover_image_path_before + "\"")
	cover_image_path_after := hugo_dir + "static/images/" + imagename
	//fmt.Println("image path after: " + "\"" + cover_image_path_after + "\"")
	copy_cover_image := exec.Command("/bin/cp", cover_image_path_before, cover_image_path_after)
	copy_cover_image.Dir = "/"
	fmt.Println("Moving inline image to hugo directory...")
	out, err := copy_cover_image.CombinedOutput()
	if err != nil {
		fmt.Println("[ERROR] Error moving " + imagename +": ", err)
	}
	fmt.Println("Moved the image: ", out)
	frontmatterimage := "    \"image\": \"" + imagename + "\""
	hugoFrontMatter = append(hugoFrontMatter, frontmatterimage)
	// Caption for image
	var frontimagecaption []string
	frontimagecaption, i = regex_line_of_markdown(markdownfile.Contents, `##### +(.*)`, `#####`, i)
	frontmattercaption := "<p class\"front-matter-image-caption\">" + frontimagecaption[0] + "</p>"
	// Now find the headline of the article
	var title []string
	title, i = regex_line_of_markdown(markdownfile.Contents, `# +(.*)`, `#`, i)
	headline := fmt.Sprintf("%f", title)
	headline = strings.Replace(headline, `%!f(string=# `, ``, -1)
	headline = strings.Replace(headline, `)`, ``, -1)
	headline = strings.Replace(headline, `(`, ``, -1)
	headline = strings.Replace(headline, `[`, `"`, -1)
	headline = strings.Replace(headline, `]`, `"`, -1)
	headline = "    \"title\": " + headline
	hugoFrontMatter = append(hugoFrontMatter, headline)
	// Find the subtitle
	var subtitle []string
	subtitle, i = regex_line_of_markdown(markdownfile.Contents, `# +(.*)`, `##`, i)
	description := fmt.Sprintf("%f", subtitle)
	description = strings.Replace(description, `%!f(string=# `, ``, -1)
	description = strings.Replace(description, `)`, ``, -1)
	description = strings.Replace(description, `(`, ``, -1)
	description = strings.Replace(description, `[`, `"`, -1)
	description = strings.Replace(description, `]`, `"`, -1)
	description = "    \"description\": " + description
	hugoFrontMatter = append(hugoFrontMatter, description)
	// Find the authors on the byline
	var author_names []string
	author_names, i = regex_line_of_markdown(markdownfile.Contents, `[^(####By |,and|,)](?:By | and)*?(\w+.\w+)`, `#### By`, i)
	author_list := fmt.Sprintf("%f", author_names)
	author_list = strings.Replace(author_list, `%!f(string=`, `"`, -1)
	author_list = strings.Replace(author_list, `) `, `", `, -1)
	author_list = strings.Replace(author_list, `)`, `"`, -1)
	author_list = "    \"authors\": " + author_list
	hugoFrontMatter = append(hugoFrontMatter, author_list)
	hugoFrontMatter = append(hugoFrontMatter, "}")
	hugoFrontMatter = append(hugoFrontMatter, "")
	hugoFrontMatter = append(hugoFrontMatter, frontmattercaption)
	hugoFrontMatter = append(hugoFrontMatter, "")
	// Delete deprecated lines
	var deleteline sync.WaitGroup
	for k := 0; k < i; k++ {
		deleteline.Add(1)
		deleteLineWrapper(md_file_path, &deleteline)
		deleteline.Wait()
	}
	// Now write the hugo front-matter to the file
	var prepend sync.WaitGroup
	prepend.Add(1)
	markdownfile = NewMarkdownFile(md_file_path)
	err = markdownfile.readMarkdownLines()
	if err != nil {
		fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
	}
	go prependWrapper(hugoFrontMatter, md_file_path, &prepend)
	prepend.Wait()
	// For-loop through the rest of the document looking for in-line images
	// in-line headers are taken care of on frontend by hugo's theme
	// in-line captions are taken care of on frontend by hugo's theme
	var rewriteimageline sync.WaitGroup
	for j := 0; j < len(markdownfile.Contents); j++ {
		markdownfile = NewMarkdownFile(md_file_path)
		err = markdownfile.readMarkdownLines()
		if err != nil {
			fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
		}
		if strings.Index(markdownfile.Contents[j], `<img src=`) >= 0 {
			rewriteimageline.Add(1)
			re2 := regexp.MustCompile(`(\w+.png)`)
			inline_image := re2.FindAllString(markdownfile.Contents[j], -1)
			inline_image_path_before := path.Dir(path.Dir(docx_file_path)) + "/" + inline_image[1]
			inline_image_path_after := hugo_dir + "static/images/" + inline_image[1]
			copy_image := exec.Command("/bin/cp", inline_image_path_before, inline_image_path_after)
			copy_image.Dir = "/"
			fmt.Println("Moving inline image to hugo directory...")
			out, err := copy_image.Output()
			if err != nil {
				fmt.Println("[ERROR] Error moving"+inline_image[1]+": ", err)
				return
			}
			fmt.Println("Moving the image: ", out)
			fmt.Println("Done moving " + inline_image[1])
			// Before writing the new line make sure that the path points to the production directory
			inline_image_path_after = nginx_dir + "public/images/" + inline_image[1]
			fmt.Println("Writing a new inline-image path for " + md_file_path)
			// Use the image caption as the alt text for the inline-image
			regex_alt_text := regexp.MustCompile(`##### +(.*)`)
			alt_texts := regex_alt_text.FindAllString(markdownfile.Contents[j + 2], -1)
			alt_text := strings.Replace(alt_texts[0], `##### `, ``, -1)
			// Rewrite the inline image to have a css class called inline-image
			newimageinline := "<img src= \"" + inline_image_path_after + "\" alt=\"" + alt_text + "\" class=\"inline-image\">"
			go rewriteMarkdownLine(j, newimageinline, md_file_path, &rewriteimageline)
			rewriteimageline.Wait()
			j = j + 2
		}
	}
	fmt.Println("Done!")
	front_matter.Done()
}

// Use hugo to compile the markdown files into html and then serve with hugo or with nginx
func compile_and_serve_hugo_site(hugo_dir string, prod_dir string, use_hugo bool, wg *sync.WaitGroup) {
}

func main() {
	// Get the user's home directory
	usr, err := user.Current()
	HOME := usr.HomeDir
	if err != nil {
		fmt.Println("[ERROR] driveraker could not get the user's home directory")
	}
	// Set the driveraker config path
	driveraker_config := HOME + "/.config/driveraker/config"
	// Read the driveraker config
	conf_message := make(chan string)
	var conf sync.WaitGroup
	conf.Add(1)
	go read_cfg(driveraker_config, &conf, conf_message)
	// Set the configured paths
	drive_sync_dir := <-conf_message
	drive_remote_dir := <-conf_message
	hugo_post_dir := <-conf_message
	nginx_dir := <-conf_message
	conf.Wait()
	// Sync Google Drive
	docx_paths_message := make(chan []string)
	var drive_sync sync.WaitGroup
	drive_sync.Add(1)
	go sync_google_drive(drive_sync_dir, drive_remote_dir, &drive_sync, docx_paths_message)
	docx_file_paths := <-docx_paths_message
	fmt.Printf("docx file paths: %s \n", docx_file_paths)
	drive_sync.Wait()
	// Convert the docx files into markdown files
	var pandoc sync.WaitGroup
	pandoc.Add(len(docx_file_paths))
	var markdown_paths []string
	fmt.Println("Converting synced docx files into markdown files...")
	for i := 0; i < len(docx_file_paths); i++ {
		docx_file_path := docx_file_paths[i]
		docx_file_path = strings.Replace(docx_file_path, ` to '`, ``, -1)
		docx_file_path = strings.Replace(docx_file_path, `docx'`, `docx`, -1)
		docx_file_paths[i] = docx_file_path
		fmt.Println("Converting " + docx_file_path)
		name_regex := regexp.MustCompile(`(\w+)(?:.docx)`)
		name := name_regex.FindAllString(docx_file_path, -1)
		markdown_path := hugo_post_dir + "content/articles/" + name[0] + ".md"
		markdown_paths = append(markdown_paths, markdown_path)
		go convert_to_markdown_with_pandoc(docx_file_path, markdown_path, &pandoc)
	}
	pandoc.Wait()
	// Add hugo front-matter to the files
	var frontmatter sync.WaitGroup
	frontmatter.Add(len(markdown_paths))
	fmt.Println("Adding hugo front-matter to markdown files...")
	for i := 0; i < len(markdown_paths); i++ {
		go read_markdown_write_hugo_headers(markdown_paths[i], docx_file_paths[i], hugo_post_dir, nginx_dir, &frontmatter)
	}
	frontmatter.Wait()
}
