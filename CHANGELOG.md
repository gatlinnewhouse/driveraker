@project: driveraker 
@author: Gatlin Newhouse <gatlin.newhouse@gmail.com>  
@homepage: 

.devist

# Unreleased (2017-07-31):
- [Fixed] #6 (a1dd045)
    driveraker can now generate markdown files with hugo front-matter from synced docx files. Although driveraker does not use the hashtable to check for already processed files yet, but I can work on that next week. Also will need to do tests with: modifying a synced document, adding documents, the hashtable, and follow the rest of my schedule.
- [Fixed] error moving images, need to fix error prepending hugo front-matter (40930ca)
- [Changed] to JSON front-matter (d2c91ec)
- [Added] lastnames (b621f96)
    Gonna need them
- [Added] newest submodule with library commits (f1e6847)
- [Removed] submodules - fingers crossed (2f214f1)
- [Removed] faulty submodule (4991483)
- [Added] the prepend to file code (8cfc4fe)
- [Added] Hugo headers prepend function. Currently writing this commit from my phone, I have commented in a link for prepending strings to a file (0c64082)
- [Added] regex expression to match all the paths of synced docx files (01a5b60)
- [Added] script to delete the unnecessary files which are synced by drive (0539bd6)
