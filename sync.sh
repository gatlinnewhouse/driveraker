#!/bin/bash

# Pull/Sync the files
drive pull -desktop-links=false -export docx $REMOTE_DIR

# Convert files to Markdown
pandoc --atx-headers --smart --normalize --email-obfuscation=references --mathjax -t markdown_strict -o $FILE.md $FILE.html
