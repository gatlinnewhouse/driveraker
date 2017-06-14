#!/bin/bash

# Pull/Sync the files
drive pull -desktop-links=false -export html $REMOTE_DIR

# Convert files to Markdown
pandoc --atx-headers --smart --normalize --email-obfuscation=references --mathjax -t markdown_strict -o $FILE.md $FILE.html

# Convert newly converted markdown file to html since there are still some html tags
mv $FILE.md $FILE.html

