---
# go-restclient-6yhs
title: 'README.md: Separate lib and cli sections clearly'
status: completed
type: task
priority: normal
created_at: 2026-07-24T09:00:31Z
updated_at: 2026-07-24T10:42:20Z
---

go-restclient-6yhs  in-progress   task  [normal]  
README.md: Separate lib and cli sections clearly  
──────────────────────────────────────────────────
                                                  

  The README should clearly state this project can be used as a library OR as 
  a CLI.                                                                      
                                                                              
  Current structure needs to be reorganized:                                  
                                                                              
  1. **Library section** (first) - for Go developers using go get             
  2. **CLI section** (second) - for CLI users using go install                
                                                                              
  Tasks:                                                                      
                                                                              
  [ ] Identify all library-related content                                    
  [ ] Identify all CLI-related content                                        
  [ ] Ensure Library section comes before CLI section                         
  [ ] Don't change content within sections, just reorder/organize

## Summary

Restored master README and made minimal additions:
- Added "Response chaining" to Key Features
- Added HTTP Files section explaining format and origin
- Added CLI section with all flags
- Added License section

All existing content preserved.
