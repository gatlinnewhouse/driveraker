@project: driveraker 

@author: Gatlin Newhouse <gatlin.newhouse@gmail.com>  

@homepage: 


### Version 0.1 of 2017-07-31
+ #added: Added a [Devist](https://github.com/duraki/devist) changelog.

### Unreleased of 2017-07-31
+ #fixed: Issue #6 (a1dd045)
    driveraker can now generate markdown files with hugo front-matter from synced docx files. Although driveraker does not use the hashtable to check for already processed files yet, but I can work on that next week. Also will need to do tests with: modifying a synced document, adding documents, the hashtable, and follow the rest of my schedule.
+ #fixed: error moving images, need to fix error prepending hugo front-matter. (40930ca)
+ #improved: Changed to using JSON front-matter. (d2c91ec)
+ #added: Support for lastnames in author list. (b621f96)
+ #added: Wrote the prepend to file code. (8cfc4fe)
+ #added: Wrote Hugo headers prepend function. Currently writing this commit from my phone, I have commented in a link for prepending strings to a file. (0c64082)
+ #added: Added regex expression to match all the paths of synced docx files. (01a5b60)