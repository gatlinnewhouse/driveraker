#!/bin/bash
# delete all empty files and .desktop files for synced documents from Google Drive
find . -type f -empty -delete
find . -name "*.desktop" -delete
