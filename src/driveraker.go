package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strings"
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
	mu    sync.RWMutex
	paths map[string]string
	count int
	save  chan record
}

type record struct {
	key, path string
}

// Use md5 hash sums for the filepaths
func md5hash(text, DriveSyncDirectory string) string {
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
func (s *PathStore) Get(key, path *string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, okay := s.paths[*key]; okay {
		*path = p
		return nil
	}
	return errors.New("Key not found")
}

// Write a new path to the hashtable for an known key
func (s *PathStore) Set(key, path *string) error {
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
		*key = md5hash(fmt.Sprintf("%s", path), fmt.Sprintf("%s", DriveSyncDirectory))
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
	ProductionDirectory        string
}

// Read the configuration JSON file in order to get some settings and directories
func readConfig(filename string, conf *sync.WaitGroup, confMessage chan string) {
	fmt.Println("Reading configuration...")
	file, _ := os.Open(filename)
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("[ERROR] Error reading the JSON confguration: ", err)
		return
	}
	driveSyncDirectory := fmt.Sprintf(configuration.DriveSyncDirectory)
	confMessage <- driveSyncDirectory
	driveRemoteDirectory := fmt.Sprintf(configuration.GoogleDriveRemoteDirectory)
	confMessage <- driveRemoteDirectory
	hugoPostDirectory := fmt.Sprintf(configuration.HugoPostDirectory)
	confMessage <- hugoPostDirectory
	productionDirectory := fmt.Sprintf(configuration.ProductionDirectory)
	confMessage <- productionDirectory
	fmt.Println("Finished reading configuration!")
	conf.Done()
}

// Sync google drive remote folder to the configured local directory.
// Then send the output from drive CLI to a function to intepret the output
// by stripping the full output down to an array of string paths to docx files.
func syncGoogleDrive(syncDirectory string, driveRemoteDirectory string, drive_sync *sync.WaitGroup, docxPathsMessage chan []string) {
	syncGDrive := new(sync.WaitGroup)
	output := make(chan string)
	filePaths := make(chan []string)
	sync := exec.Command("/usr/bin/drive", "pull", "-no-prompt", "-desktop-links=false", "-export", "docx", driveRemoteDirectory)
	sync.Dir = syncDirectory
	fmt.Println("Syncing Google Drive...")
	out, err := sync.Output()
	if err != nil {
		fmt.Println("[ERROR] Error syncing Google Drive: ", err)
		return
	}
	fmt.Printf("drive: " + string(out))
	fmt.Println("Done syncing!")
	syncGDrive.Add(1)
	go interpretDriveOutput(syncGDrive, output, filePaths)
	output <- string(out)
	docxPaths := <-filePaths
	syncGDrive.Wait()
	docxPathsMessage <- docxPaths
	drive_sync.Done()
}

// Find all Exported file paths via a regex expression and then add them to an array
func interpretDriveOutput(syncGDrive *sync.WaitGroup, output chan string, filePaths chan []string) {
	fmt.Println("Interpreting command line output...")
	results := <-output
	re := regexp.MustCompile(`[^'](?:to ')(.*?)'`)
	matches := re.FindAllString(results, -1)
	fmt.Printf("File paths: %s \n", matches)
	filePaths <- matches
	fmt.Println("Done!")
	syncGDrive.Done()
}

// Convert from docx to markdown with pandoc
func convertToMarkdownWithPandoc(docxFilePath string, markdownFilePath string, pandoc *sync.WaitGroup) {
	convert := exec.Command("/usr/bin/pandoc", "--atx-headers", "--smart", "--normalize", "--email-obfuscation=references", "--mathjax", "-t", "markdown_strict", "-o", markdownFilePath, docxFilePath)
	convert.Dir = "/"
	out, err := convert.CombinedOutput()
	if err != nil {
		fmt.Println("[ERROR] Error converting files to markdown with pandoc: ", err)
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

func prependWrapper(content []string, markdownFilePath string, prepend *sync.WaitGroup) {
	err := NewMarkdownFile(markdownFilePath).Prepend(content)
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

func deleteLineWrapper(markdownFilePath string, deleteline *sync.WaitGroup) {
	f, err := os.OpenFile(markdownFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("[ERROR] Error opening file: ", err)
	}
	defer f.Close()
	line, err := deleteLine(f)
	if err != nil {
		fmt.Println("[ERROR] Error deleting a line: ", err)
	}
	fmt.Printf("Deleted line: %s from %s\n", string(line), markdownFilePath)
	f.Close()
	deleteline.Done()
}

/* =============================== */
/* End of modified popline.go code */
/* =============================== */

// Rewrite a line in a file
func rewriteMarkdownLine(line int, replacement string, markdownFilePath string, rewritemarkdown *sync.WaitGroup) {
	input, err := ioutil.ReadFile(markdownFilePath)
	if err != nil {
		fmt.Println("[ERROR] Error opening the file", err)
	}
	contents := strings.Split(string(input), "\n")
	contents[line] = replacement
	output := strings.Join(contents, "\n")
	err = ioutil.WriteFile(markdownFilePath, []byte(output), 0644)
	if err != nil {
		fmt.Println("[ERROR] There was an error writing the file")
	}
	rewritemarkdown.Done()
}

// General function for regex
func regexLineOfMarkdown(contents []string, regex string, variable string, line int) (value []string, lineNumber int) {
	if strings.Index(contents[line], variable) >= 0 {
		re := regexp.MustCompile(regex)
		value = re.FindAllString(contents[line], -1)
		// if we find it, move down a line
		lineNumber = line + 2
		return
	}
	value = append(value, "")
	lineNumber = line
	// didn't find anything, then leave blank and do not iterate the line number
	return
}

// Read markdown document and write the hugo headers to the beginning of the document
func readMarkdownWriteHugoHeaders(markdownFilePath string, docxFilePath string, hugoDirectory string, productionDirectory string, front_matter *sync.WaitGroup) {
	markdownfile := NewMarkdownFile(markdownFilePath)
	err := markdownfile.readMarkdownLines()
	if err != nil {
		fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
	}
	// Read and then rewrite the line read according to what value it should be
	var i int                    // The number of driveraker front matter lines
	i = 0                        // For the reading line, start at 0
	var hugoFrontMatter []string // Add all hugo front matter to this string slice
	hugoFrontMatter = append(hugoFrontMatter, "{")
	// Find DRVRKR\_TAGS
	var tags []string
	tags, i = regexLineOfMarkdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_TAGS)](\w+)`, "DRVRKR\\_TAGS", i)
	tagsList := fmt.Sprintf("%f", tags)
	tagsList = strings.Replace(tagsList, `%!f(string= `, `"`, -1)
	tagsList = strings.Replace(tagsList, `) `, `", `, -1)
	tagsList = strings.Replace(tagsList, `)`, `"`, -1)
	tagsList = "    \"tags\": " + tagsList + ","
	hugoFrontMatter = append(hugoFrontMatter, tagsList)
	// Now find the DRVRKR\_CATEGORIES
	var categories []string
	categories, i = regexLineOfMarkdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_CATEGORIES)](\w+)`, "DRVRKR\\_CATEGORIES", i)
	categoriesList := fmt.Sprintf("%f", categories)
	categoriesList = strings.Replace(categoriesList, `%!f(string= `, `"`, -1)
	categoriesList = strings.Replace(categoriesList, `) `, `", `, -1)
	categoriesList = strings.Replace(categoriesList, `)`, `"`, -1)
	categoriesList = "    \"categories\": " + categoriesList + ","
	hugoFrontMatter = append(hugoFrontMatter, categoriesList)
	// Draft status
	hugoFrontMatter = append(hugoFrontMatter, "    \"draft\": \"false\",")
	// Now find the DRVRKR\_PUB\_DATE
	var publicationyearmonthdate []string
	publicationyearmonthdate, i = regexLineOfMarkdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_PUB\\\_DATE)](\w+)`, "DRVRKR\\_PUB\\_DATE", i)
	publicationDate := fmt.Sprintf("%f", publicationyearmonthdate)
	publicationDate = strings.Replace(publicationDate, `%!f(string= `, ``, -1)
	publicationDate = strings.Replace(publicationDate, `)`, ``, -1)
	publicationDate = strings.Replace(publicationDate, `[`, `"`, -1)
	publicationDate = strings.Replace(publicationDate, `]`, `"`, -1)
	publicationDate = strings.Replace(publicationDate, ` `, `-`, -1)
	hugoFrontMatter = append(hugoFrontMatter, "    \"date\": " + publicationDate + ",")
	hugoFrontMatter = append(hugoFrontMatter, "    \"publishDate\": " + publicationDate + ",")
	// Now find the DRVRKR\_UPDATE\_DATE
	var updateyearmonthdate []string
	updateyearmonthdate, i = regexLineOfMarkdown(markdownfile.Contents, `[^\\\_:,\n]*?[^(DRVRKR\\\_UPDATE\\\_DATE)](\w+)`, "DRVRKR\\_UPDATE\\_DATE", i)
	modificationDate := fmt.Sprintf("%f", updateyearmonthdate)
	modificationDate = strings.Replace(modificationDate, `%!f(string= `, ``, -1)
	modificationDate = strings.Replace(modificationDate, `)`, ``, -1)
	modificationDate = strings.Replace(modificationDate, `[`, `"`, -1)
	modificationDate = strings.Replace(modificationDate, `]`, `"`, -1)
	modificationDate = strings.Replace(modificationDate, ` `, `-`, -1)
	modificationDate = "    \"lastmod\": " + modificationDate + ","
	hugoFrontMatter = append(hugoFrontMatter, modificationDate)
	// Now find the cover photo for the article
	var imagenames []string
	imagenames, i = regexLineOfMarkdown(markdownfile.Contents, `(\w+.png)`, `<img src=`, i)
	imagename := imagenames[1]
	coverImagePathBefore := path.Dir(path.Dir(docxFilePath)) + "/" + imagename
	//fmt.Println("image path before: " + "\"" + coverImagePathBefore + "\"")
	coverImagePathAfter := hugoDirectory + "static/images/" + imagename
	//fmt.Println("image path after: " + "\"" + coverImagePathAfter + "\"")
	copyCoverImage := exec.Command("/bin/cp", coverImagePathBefore, coverImagePathAfter)
	copyCoverImage.Dir = "/"
	fmt.Println("Moving inline image to hugo directory...")
	out, err := copyCoverImage.CombinedOutput()
	if err != nil {
		fmt.Println("[ERROR] Error moving " + imagename + ": ", err)
	}
	fmt.Println("Moved the image: ", out)
	frontmatterimage := "    \"image\": \"" + imagename + "\","
	hugoFrontMatter = append(hugoFrontMatter, frontmatterimage)
	// Caption for image
	var frontimagecaption []string
	frontimagecaption, i = regexLineOfMarkdown(markdownfile.Contents, `##### +(.*)`, `#####`, i)
	frontmattercaption := "<p class=\"front-matter-image-caption\">" + frontimagecaption[0] + "</p>"
	// Now find the headline of the article
	var title []string
	title, i = regexLineOfMarkdown(markdownfile.Contents, `# +(.*)`, `#`, i)
	headline := fmt.Sprintf("%f", title)
	headline = strings.Replace(headline, `%!f(string=# `, ``, -1)
	headline = strings.Replace(headline, `)`, ``, -1)
	headline = strings.Replace(headline, `(`, ``, -1)
	headline = strings.Replace(headline, `[`, `"`, -1)
	headline = strings.Replace(headline, `]`, `"`, -1)
	headline = "    \"title\": " + headline + ","
	hugoFrontMatter = append(hugoFrontMatter, headline)
	// Find the subtitle
	var subtitle []string
	subtitle, i = regexLineOfMarkdown(markdownfile.Contents, `# +(.*)`, `##`, i)
	description := fmt.Sprintf("%f", subtitle)
	description = strings.Replace(description, `%!f(string=# `, ``, -1)
	description = strings.Replace(description, `)`, ``, -1)
	description = strings.Replace(description, `(`, ``, -1)
	description = strings.Replace(description, `[`, `"`, -1)
	description = strings.Replace(description, `]`, `"`, -1)
	description = "    \"description\": " + description + ","
	hugoFrontMatter = append(hugoFrontMatter, description)
	// Find the authors on the byline
	var author_names []string
	author_names, i = regexLineOfMarkdown(markdownfile.Contents, `[^(####By |,and|,)](?:By | and)*?(\w+.\w+)`, `#### By`, i)
	authorList := fmt.Sprintf("%f", author_names)
	authorList = strings.Replace(authorList, `%!f(string=`, `"`, -1)
	authorList = strings.Replace(authorList, `) `, `", `, -1)
	authorList = strings.Replace(authorList, `)`, `"`, -1)
	authorList = "    \"authors\": " + authorList
	hugoFrontMatter = append(hugoFrontMatter, authorList)
	hugoFrontMatter = append(hugoFrontMatter, "}")
	hugoFrontMatter = append(hugoFrontMatter, "")
	hugoFrontMatter = append(hugoFrontMatter, frontmattercaption)
	hugoFrontMatter = append(hugoFrontMatter, "")
	// Delete deprecated lines
	var deleteline sync.WaitGroup
	for k := 0; k < i; k++ {
		deleteline.Add(1)
		deleteLineWrapper(markdownFilePath, &deleteline)
		deleteline.Wait()
	}
	// Now write the hugo front-matter to the file
	var prepend sync.WaitGroup
	prepend.Add(1)
	markdownfile = NewMarkdownFile(markdownFilePath)
	err = markdownfile.readMarkdownLines()
	if err != nil {
		fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
	}
	go prependWrapper(hugoFrontMatter, markdownFilePath, &prepend)
	prepend.Wait()
	// For-loop through the rest of the document looking for in-line images
	// in-line headers are taken care of on frontend by hugo's theme
	// in-line captions are taken care of on frontend by hugo's theme
	var rewriteimageline sync.WaitGroup
	for j := 0; j < len(markdownfile.Contents); j++ {
		markdownfile = NewMarkdownFile(markdownFilePath)
		err = markdownfile.readMarkdownLines()
		if err != nil {
			fmt.Println("[ERROR] Error reading lines from the markdown file: ", err)
		}
		if strings.Index(markdownfile.Contents[j], `<img src=`) >= 0 {
			rewriteimageline.Add(1)
			re2 := regexp.MustCompile(`(\w+.png)`)
			inline_image := re2.FindAllString(markdownfile.Contents[j], -1)
			inline_image_path_before := path.Dir(path.Dir(docxFilePath)) + "/" + inline_image[1]
			inline_image_path_after := hugoDirectory + "static/images/" + inline_image[1]
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
			inline_image_path_after = productionDirectory + "public/images/" + inline_image[1]
			fmt.Println("Writing a new inline-image path for " + markdownFilePath)
			// Use the image caption as the alt text for the inline-image
			regex_alt_text := regexp.MustCompile(`##### +(.*)`)
			alt_texts := regex_alt_text.FindAllString(markdownfile.Contents[j+2], -1)
			alt_text := strings.Replace(alt_texts[0], `##### `, ``, -1)
			// Rewrite the inline image to have a css class called inline-image
			newimageinline := "<img src=\"" + inline_image_path_after + "\" alt=\"" + alt_text + "\" class=\"inline-image\">"
			go rewriteMarkdownLine(j, newimageinline, markdownFilePath, &rewriteimageline)
			rewriteimageline.Wait()
			j = j + 2
		}
	}
	fmt.Println("Done!")
	front_matter.Done()
}

// Use hugo to compile the markdown files into html and then move the files to the production directory, i.e. where nginx or apache serve files
// Make sure to chown or chmod the production directory before running driveraker
func compile_and_serve_hugo_site(hugoDirectory string, prod_dir string, serve *sync.WaitGroup) {
	compile := exec.Command("/usr/bin/hugo")
	compile.Dir = hugoDirectory
	out, err := compile.Output()
	if err != nil {
		fmt.Println("[ERROR] Error compiling a website with hugo: ", err)
	}
	fmt.Println("hugo: ", string(out))
	copyCompiledSite := exec.Command("/bin/cp", "-r", "-u", hugoDirectory + "public/*", prod_dir)
	copyCompiledSite.Dir = "/"
	fmt.Println("Moving compiled hugo site to the production directory...")
	out, err = copyCompiledSite.Output()
	if err != nil {
		fmt.Println("[ERROR] Error moving hugo compiled site to production directory: ", err)
		return
	}
	fmt.Println("Moving the compiled hugo site: ", out)
	serve.Done()
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
	confMessage := make(chan string)
	var conf sync.WaitGroup
	conf.Add(1)
	go readConfig(driveraker_config, &conf, confMessage)
	// Set the configured paths
	driveSyncDirectory := <-confMessage
	driveRemoteDirectory := <-confMessage
	hugoPostDirectory := <-confMessage
	productionDirectory := <-confMessage
	conf.Wait()
	// Sync Google Drive
	docxPathsMessage := make(chan []string)
	var drive_sync sync.WaitGroup
	drive_sync.Add(1)
	go syncGoogleDrive(driveSyncDirectory, driveRemoteDirectory, &drive_sync, docxPathsMessage)
	docxFilePaths := <-docxPathsMessage
	fmt.Printf("docx file paths: %s \n", docxFilePaths)
	drive_sync.Wait()
	// Convert the docx files into markdown files
	var pandoc sync.WaitGroup
	pandoc.Add(len(docxFilePaths))
	var markdownPaths []string
	fmt.Println("Converting synced docx files into markdown files...")
	for i := 0; i < len(docxFilePaths); i++ {
		docxFilePath := docxFilePaths[i]
		docxFilePath = strings.Replace(docxFilePath, ` to '`, ``, -1)
		docxFilePath = strings.Replace(docxFilePath, `docx'`, `docx`, -1)
		docxFilePaths[i] = docxFilePath
		fmt.Println("Converting " + docxFilePath)
		name_regex := regexp.MustCompile(`(\w+)(?:.docx)`)
		name := name_regex.FindAllString(docxFilePath, -1)
		markdown_path := hugoPostDirectory + "content/articles/" + name[0] + ".md"
		markdownPaths = append(markdownPaths, markdown_path)
		go convertToMarkdownWithPandoc(docxFilePath, markdown_path, &pandoc)
	}
	pandoc.Wait()
	// Add hugo front-matter to the files
	var frontmatter sync.WaitGroup
	frontmatter.Add(len(markdownPaths))
	fmt.Println("Adding hugo front-matter to markdown files...")
	for i := 0; i < len(markdownPaths); i++ {
		go readMarkdownWriteHugoHeaders(markdownPaths[i], docxFilePaths[i], hugoPostDirectory, productionDirectory, &frontmatter)
	}
	frontmatter.Wait()
	// Serve the website by compiling the site with hugo and moving it to the production directory
	var serveWebsite sync.WaitGroup
	serveWebsite.Add(1)
	go compile_and_serve_hugo_site(hugoPostDirectory, productionDirectory, &serveWebsite)
	serveWebsite.Wait()
	// Send back a success message and code
	fmt.Println("driveraker successfully synced, converted, and compiled Google Documents into a website")
	fmt.Println("Thanks to other open source projects like:")
	fmt.Println("* Emmanuel Odeke's drive command line client for Google Drive")
	fmt.Println("* John MacFarlane's pandoc file converter")
	fmt.Println("* And many more...")
	os.Exit(0)
}
