package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strings"
	"sync"
)

/*
Hashtable code forked from
https://github.com/jackfhebert/hashtable
*/

// Not publicly visible since it is our internal wrapper.
type linkedListNode struct {
	// The value of this node.
	// Once I figure out interfaces, that will go here.
	Value interface{} `json:"Value"`
	// Pointer to the next node in the list.
	Next *linkedListNode `json:"Next"`
}

// Exposed - this is the struct to use.
type LinkedList struct {
	// The first node in the list.
	First *linkedListNode `json:"First"`
	// The last item in the list.
	// This makes adding to the list fast, but isn't strictly
	// needed.
	Last *linkedListNode `json:"Last"`
	// How many items are in the list. This is mostly
	// for the size helper and not strictly needed.
	Size int `json:"Size"`
}

func NewLinkedList() *LinkedList {
	return &LinkedList{nil, nil, 0}
}

func (list *LinkedList) SizeOf() int {
	return list.Size
}

func (list *LinkedList) AddItem(item interface{}) {
	list.Size += 1
	node := &linkedListNode{item, nil}
	if list.First == nil {
		list.First = node
	}
	if list.Last != nil {
		list.Last.Next = node
	}
	list.Last = node
}

func (list *LinkedList) RemoveItem(item interface{}) {
	// Track the previous node from the iterator for updating
	// pointers between nodes.
	var prevNode *linkedListNode
	prevNode = nil
	for currNode := list.First; currNode != nil; currNode = currNode.Next {
		if currNode.Value == item {
			// Update the list metadata.
			list.Size -= 1
			if currNode == list.First {
				list.First = currNode.Next
			}
			if currNode == list.Last {
				list.Last = prevNode
			}
			// Update the nodes.
			if prevNode != nil {
				prevNode.Next = currNode.Next
			}

			// All done here.
			return
		}

		// Keep iterating through the list. I could probably assign
		// this in the for-loop definition above.  
		prevNode = currNode
	}
}

func (list *LinkedList) Items() []*interface{} {
	items := make([]*interface{}, list.Size)

	for i, currNode := 0, list.First; currNode != nil; currNode = currNode.Next {
		items[i] = &currNode.Value
		i += 1
	}
	return items
}

var (
	fillRate int = 10
)

type HashTable struct {
	// Number of items in table
	Size int `json:"Size"`
	// Maximum number of items in table
	Capacity int `json:"Capacity"`
	// Array of linkedlist pointers
	Items []*LinkedList `json:"Items"`
}

// Internal helper to wrap the key and value which were added to
// the hashtable. This is the value stored in the linked lists
// per bucket.
type tableItem struct {
	Key   string `json:"Key"`
	Value interface{} `json:"Value"`
}

// Create a new hashtable with a given number of buckets. This
// probably should have been made to specify the capacity instead.
func NewHashTableSized(size int) *HashTable {
	table := &HashTable{0, fillRate * size, make([]*LinkedList, size)}
	for i := 0; i < len(table.Items); i++ {
		table.Items[i] = nil
	}
	return table
}

// Return a default hashtable.
func NewHashTable() *HashTable {
	return NewHashTableSized(128)
}

// Make the hashtable bigger so that there are fewer items per bucket.
// This takes up more memory (ish) but reduces the number of items per
// bucket which makes the datastructure faster to use.
func (table *HashTable) resizeTable() {
	next := NewHashTableSized(2 * len(table.Items))
	for _, list := range table.Items {
		if list != nil {
			for _, item := range list.Items() {
				if parsed, ok := (*item).(tableItem); ok {
					next.AddItem(parsed.Key, parsed.Value)
				} else {
					fmt.Println("failed to parse item in resize", item)
				}
			}
		}
	}
	table = next
}

// Helper function to take the key string and determine which bucket
// the item should be placed in.
func getIndex(key string, max int) int {
	hash := adler32.New()
	hash.Write([]byte(key))
	digest := hash.Sum32()
	return int(digest) % max
}

// Shorten the values for paths by getting rid of the DriveSyncDirectory string
func shortenPath(fullpath, DriveSyncDirectory string) string {
	r := strings.NewReplacer(DriveSyncDirectory, "")
	relativePath := r.Replace(fullpath)
	return relativePath
}

// Add a key, value paid to the hash table.
func (table *HashTable) AddItem(key string, value interface{}) {
	index := getIndex(key, len(table.Items))
	if table.Items[index] == nil {
		table.Items[index] = NewLinkedList()
	}
	table.Size += 1
	table.Items[index].AddItem(tableItem{key, value})
	if table.Size > table.Capacity {
		table.resizeTable()
	}
}

// Remove all instances of a key from the table.
func (table *HashTable) RemoveKey(key string) bool {
	index := getIndex(key, len(table.Items))
	if table.Items[index] != nil {
		for _, item := range table.Items[index].Items() {
			if parsed, ok := (*item).(tableItem); ok {
				if parsed.Key == key {
					table.Items[index].RemoveItem(item)
					table.Size -= 1
				}
			}
		}
	}
	return false
}

// Determine if a key is contained in the hash table.
func (table *HashTable) ContainsKey(key string) bool {
	fmt.Println("Checking for key:", key)
	index := getIndex(key, len(table.Items))
	if table.Items[index] != nil {
		for _, item := range table.Items[index].Items() {
			if parsed, ok := (*item).(tableItem); ok {
				if parsed.Key == key {
					fmt.Println(parsed.Key)
					return true
				}
			} else {
				fmt.Println("failed to parse item in contains", *item)
			}
		}
	}
	return false
}

func (table *HashTable) GetValue(key string) interface{} {
	fmt.Println("Checking for key:", key)
	index := getIndex(key, len(table.Items))
	if table.Items[index] != nil {
		for _, item := range table.Items[index].Items() {
			if parsed, ok := (*item).(tableItem); ok {
				if parsed.Key == key {
					return parsed.Value
				}
			} else {
				fmt.Println("failed to parse item in contains", *item)
			}
		}
	}
	return nil
}

func (table *HashTable) SaveHashTable(filePath string) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("[ERROR] Error opening hashtable file: ", err)
	}
	defer f.Close()
	b, err := json.Marshal(table)
	if err != nil {
		fmt.Println("[ERROR] Error saving hashtable as JSON: ", err)
	}
	f.Write(b)
	f.Close()
}

func ReadHashTable(filePath string) (table *HashTable) {
	hashtable, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("[ERROR] Error opening hashtable: ", err)
	}
	err = json.Unmarshal(hashtable, &table)
	if err != nil {
		fmt.Println("[ERROR] Error reading hashtable: ", err)
	}
	return table
}

/*
End of forked Hashtable code
*/

// The configuration file struct
type Configuration struct {
	DriveSyncDirectory          string
	GoogleDriveRemoteDirectory  string
	HugoPostDirectory           string
	ProductionDirectory         string
	HashtablePath							  string
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
	confMessage <- fmt.Sprintf(configuration.DriveSyncDirectory)
	confMessage <- fmt.Sprintf(configuration.GoogleDriveRemoteDirectory)
	confMessage <- fmt.Sprintf(configuration.HugoPostDirectory)
	confMessage <- fmt.Sprintf(configuration.ProductionDirectory)
	confMessage <- fmt.Sprintf(configuration.HashtablePath)
	fmt.Println("Finished reading configuration!")
	conf.Done()
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Sync google drive remote folder to the configured local directory.
// Then send the output from drive CLI to a function to intepret the output
// by stripping the full output down to an array of string paths to docx files.
func syncGoogleDrive(syncDirectory string, driveRemoteDirectory string, databasePath string, driveSync *sync.WaitGroup, docxPathsMessage chan []string) {
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
	go interpretDriveOutput(syncGDrive, databasePath, syncDirectory, output, filePaths)
	output <- string(out)
	docxPaths := <-filePaths
	syncGDrive.Wait()
	docxPathsMessage <- docxPaths
	driveSync.Done()
}

// Look up paths in hashtable
// Already in hashtable, then remove from the array
// Unless it is a modified document
// Otherwise add the new paths to the hashtable and forward them back to the main function
func alreadySyncedAndCompiled(matches []string, driveSyncDirectory string, hashTablePath string) []string {
	var hashTable *HashTable
	fmt.Println("Checking if hashtable already exists...")
	hashTableExists, err := exists(hashTablePath)
	// if a hashtable does not exist then make a new one
	if hashTableExists == false {
		if err != nil {
			fmt.Println("[ERROR] Error checking for hashtable: ", err)
		}
		hashTable = NewHashTableSized(len(matches))
	} else {
		// try to open the already existing hashtable
		hashTable = ReadHashTable(hashTablePath)
	}
	fmt.Println("Looking for already synced documents...")
	var alreadySynced bool
	matchesLength := len(matches)
	for i := 0; i < matchesLength; i++ {
		alreadySynced = hashTable.ContainsKey(shortenPath(matches[i], driveSyncDirectory))
		if alreadySynced == true {
			matches = append(matches[:i], matches[i+1:]...)
			matchesLength = len(matches)
			i--
		} else {
			hashTable.AddItem(shortenPath(matches[i], driveSyncDirectory), shortenPath(matches[i], driveSyncDirectory))
		}
	}
	hashTable.SaveHashTable(hashTablePath)
	return matches
}

// Find all modified documents and make sure to compile them by adding them to a string array
func findModifiedDocuments(result string) (modifiedDocuments []string) {
	fmt.Println("Looking for modified documents...")
	re := regexp.MustCompile(`M (\/.*)`)
	values := re.FindAllString(result, -1)
	var i int
	for i = 0; i < len(values); i++ {
		value := fmt.Sprintf("%f", values[0])
		value = strings.Replace(value, `%!f(string=M `, ``, -1)
		value = strings.Replace(value, `)`, ``, -1)
		filename := value[strings.LastIndex(value, "/"):len(value)]
		value = value + "_exports" + filename + ".docx"
		modifiedDocuments = append(modifiedDocuments, value)
	}
	return modifiedDocuments
}

// Find all Exported file paths via a regex expression and then add them to an array
func interpretDriveOutput(syncGDrive *sync.WaitGroup, hashtablePath string, driveSyncDirectory string, output chan string, filePaths chan []string) {
	fmt.Println("Interpreting command line output...")
	results := <-output
	re := regexp.MustCompile(`[^'](?:to ')(.*?)'`)
	matches := re.FindAllString(results, -1)
	// Make the matches into actual strings
	for i := 0; i < len(matches); i++ {
		match := matches[i]
		match = strings.Replace(match, ` to '`, ``, -1)
		match = strings.Replace(match, `docx'`, `docx`, -1)
		matches[i] = match
	}
	// Lookup entries in hashtable
	newMatches := alreadySyncedAndCompiled(matches, driveSyncDirectory, hashtablePath)
	// Find modified documents and add them to the docx paths
	modifiedDocuments := findModifiedDocuments(results)
	newMatches = append(newMatches, modifiedDocuments...)
	// Send the list of files to convert and append hugo front-matter back to the main thread
	fmt.Printf("File paths: %s \n", newMatches)
	filePaths <- newMatches
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

/* 
The following code forked from:
https://gist.github.com/toruuetani/f6aa4751a66ef65646c1a4934471396b
*/

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

/*
End of modified record.go code.

Beginning of forked popline.go code from:
https://stackoverflow.com/questions/30940190/remove-first-line-from-text-file-in-golang#30948278
*/

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

/*
End of modified popline.go code
*/

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
	hugoFrontMatter = append(hugoFrontMatter, "    \"date\": "+publicationDate+",")
	hugoFrontMatter = append(hugoFrontMatter, "    \"publishDate\": "+publicationDate+",")
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
		fmt.Println("[ERROR] Error moving "+imagename+": ", err)
	}
	fmt.Println("Moved the image: ", out)
	frontmatterimage := "    \"image\": \"" + imagename + "\","
	hugoFrontMatter = append(hugoFrontMatter, frontmatterimage)
	// Caption for image
	var frontimagecaption []string
	frontimagecaption, i = regexLineOfMarkdown(markdownfile.Contents, `##### +(.*)`, `#####`, i)
	frontimagecaption[0] = strings.Replace(frontimagecaption[0], `##### `, ``, -1)
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
	var authorNames []string
	authorNames, i = regexLineOfMarkdown(markdownfile.Contents, `[^(####By |,and|,)](?:By | and)*?(\w+.\w+)`, `#### By`, i)
	authorList := fmt.Sprintf("%f", authorNames)
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
			inlineImage := re2.FindAllString(markdownfile.Contents[j], -1)
			inlineImagePathBefore := path.Dir(path.Dir(docxFilePath)) + "/" + inlineImage[1]
			inlineImagePathAfter := hugoDirectory + "static/images/" + inlineImage[1]
			copyImage := exec.Command("/bin/cp", inlineImagePathBefore, inlineImagePathAfter)
			copyImage.Dir = "/"
			fmt.Println("Moving inline image to hugo directory...")
			out, err := copyImage.Output()
			if err != nil {
				fmt.Println("[ERROR] Error moving"+inlineImage[1]+": ", err)
				return
			}
			fmt.Println("Moving the image: ", out)
			fmt.Println("Done moving " + inlineImage[1])
			// Before writing the new line make sure that the path points to the production directory
			inlineImagePathAfter = productionDirectory + "public/images/" + inlineImage[1]
			fmt.Println("Writing a new inline-image path for " + markdownFilePath)
			// Use the image caption as the alt text for the inline-image
			regexAltText := regexp.MustCompile(`##### +(.*)`)
			altTexts := regexAltText.FindAllString(markdownfile.Contents[j+2], -1)
			altText := strings.Replace(altTexts[0], `##### `, ``, -1)
			// Rewrite the inline image to have a css class called inline-image
			newimageinline := "<img src=\"" + inlineImagePathAfter + "\" alt=\"" + altText + "\" class=\"inline-image\">"
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
func compileAndServeHugoSite(hugoDirectory string, productionDirectory string, copyHugoSiteToProductionPath string, serve *sync.WaitGroup) {
	compile := exec.Command("/usr/bin/hugo")
	compile.Dir = hugoDirectory
	out, err := compile.Output()
	if err != nil {
		fmt.Println("[ERROR] Error compiling a website with hugo: ", err)
	}
	fmt.Println("hugo: ", string(out))
	publishHugoSite := exec.Command("/bin/bash", copyHugoSiteToProductionPath, hugoDirectory + "public/", productionDirectory)
	publishHugoSite.Dir = "/"
	fmt.Println("Copying hugo compiled site to production directory...")
	out, err = publishHugoSite.Output()
	if err != nil {
		fmt.Println("[ERROR] Error copying hugo site to production: ", err)
	}
	fmt.Printf("copying hugo site to production: " + string(out))
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
  driverakerConfigPath := HOME + "/.config/driveraker/config"
	// Set the copy Hugo compiled site to production directory script path
	copyHugoSiteScript := HOME + "/.config/driveraker/copyHugoSite.sh"
	// Read the driveraker config
	confMessage := make(chan string)
	var conf sync.WaitGroup
	conf.Add(1)
	go readConfig(driverakerConfigPath, &conf, confMessage)
	// Set the configured paths
	driveSyncDirectory := <-confMessage
	driveRemoteDirectory := <-confMessage
	hugoPostDirectory := <-confMessage
	productionDirectory := <-confMessage
	hashtablePath := <-confMessage
	conf.Wait()
	// Sync Google Drive
	docxPathsMessage := make(chan []string)
	var driveSync sync.WaitGroup
	driveSync.Add(1)
	go syncGoogleDrive(driveSyncDirectory, driveRemoteDirectory, hashtablePath, &driveSync, docxPathsMessage)
	docxFilePaths := <-docxPathsMessage
	fmt.Printf("docx file paths: %s \n", docxFilePaths)
	driveSync.Wait()
	// Convert the docx files into markdown files
	var pandoc sync.WaitGroup
	pandoc.Add(len(docxFilePaths))
	var markdownPaths []string
	fmt.Println("Converting synced docx files into markdown files...")
	for i := 0; i < len(docxFilePaths); i++ {
		fmt.Println("Converting " + docxFilePaths[i])
		nameRegex := regexp.MustCompile(`(\w+)(?:.docx)`)
		name := nameRegex.FindAllString(docxFilePaths[i], -1)
		markdownPath := hugoPostDirectory + "content/articles/" + name[0] + ".md"
		markdownPaths = append(markdownPaths, markdownPath)
		go convertToMarkdownWithPandoc(docxFilePaths[i], markdownPath, &pandoc)
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
	go compileAndServeHugoSite(hugoPostDirectory, productionDirectory, copyHugoSiteScript, &serveWebsite)
	serveWebsite.Wait()
	// Send back a success message and code
	fmt.Println("driveraker successfully synced, converted, and compiled Google Documents into a website")
	fmt.Println("Thanks to other open source projects like:")
	fmt.Println("* Emmanuel Odeke's drive command line client for Google Drive")
	fmt.Println("* John MacFarlane's pandoc file converter")
	fmt.Println("* And many more...")
	os.Exit(0)
}
