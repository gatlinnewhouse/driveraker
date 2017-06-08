#!/bin/bash

####################################################################################
# This script will update all synced google drive files to:                        #
# 1. Use unix compliant names                                                      #
# 2. Delete all empty files and .desktop files that are synced                     #
# 3. Move all .docx files out of the "_exports" folders into the parent folder     #
# 4. Delete all empty folders                                                      #
####################################################################################

# Convert filenames to lowercase and replace characters recursively
if [ -z $1 ];then echo Give target directory; exit 0;fi

find "$1" -depth -name '*' | while read file ; do
        directory=$(dirname "$file")
        oldfilename=$(basename "$file")
        newfilename=$(echo "$oldfilename" | tr 'A-Z' 'a-z' | tr ' ' '_' | sed 's/_-_/-/g')
        if [ "$oldfilename" != "$newfilename" ]; then
                mv -i "$directory/$oldfilename" "$directory/$newfilename"
                echo ""$directory/$oldfilename" ---> "$directory/$newfilename""
                #echo "$directory"
                #echo "$oldfilename"
                #echo "$newfilename"
                #echo
        fi
done

# Delete all empty files and .desktop files for synced documents from Google Drive
find . -type f -empty -delete
find . -iname "*.desktop" -delete

# Move all .docx files to their parent directory
find . -iname "*.docx" -execdir mv {} .. \;

# Delete all empty folders
find . -type d -empty -delete
