@project: driveraker 

@author: Gatlin Newhouse <gatlin.newhouse@gmail.com>  

@homepage: https://gatlinnewhouse.github.io/driveraker/


### Version [v0.1-alpha](https://github.com/gatlinnewhouse/driveraker/releases/tag/v0.1-alpha) of 2017-08-07
+ #fixed: Fixed the front-matter image caption having the markdown syntax in the alt-text.
+ #fixed: Rewrote variables names to be in line with golint variable names.

### Unreleased of 2017-07-31
+ #fixed: Issue #6
    driveraker can now generate markdown files with hugo front-matter from synced docx files. Although driveraker does not use the hashtable to check for already processed files yet, but I can work on that next week. Also will need to do tests with: modifying a synced document, adding documents, the hashtable, and follow the rest of my schedule.
+ #fixed: error moving images, need to fix error prepending hugo front-matter.
+ #improved: Changed to using JSON front-matter.
+ #added: Support for lastnames in author list.
+ #added: Wrote the prepend to file code.
+ #added: Wrote Hugo headers prepend function. Currently writing this commit from my phone, I have commented in a link for prepending strings to a file.
+ #added: Added regex expression to match all the paths of synced docx files.
