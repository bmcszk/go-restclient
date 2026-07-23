---
# go-restclient-zvxw
title: 'CLI: Add -D/--define flag for command-line key:value variables'
status: completed
type: feature
priority: normal
created_at: 2026-07-23T19:50:11Z
updated_at: 2026-07-23T20:04:17Z
---

go-restclient-zvxw  in-progress   feature  [normal]           
CLI: Add -D/--define flag for command-line key:value variables
──────────────────────────────────────────────────            
                                                              

  Allow passing inline variables via CLI flags, similar to Maven's -Dkey=value
  or curl's -d.                                                               
                                                                              
  ## Use cases                                                                
                                                                              
  • Quick testing without .env files: restclient -f file.http -D env=prod -D  
  token=abc123                                                                
  • Override .env values from command line                                    
  • CI/CD injection: restclient -f deploy.http -D version=$VERSION -D         
  target=$ENV                                                                 
                                                                              
  ## Proposed syntax                                                          
                                                                              
    restclient -f file.http -D key=value -D another=value                     
    # or                                                                      
    restclient -f file.http --define key=value --define another=value         
                                                                              
  ## Implementation                                                           
                                                                              
  • Add -D / --define flag (repeatable)                                       
  • Parse as key=value pairs                                                  
  • Merge into variable resolution with highest precedence (overrides .env,   
  globals, etc.)                                                              
  • Available as {{key}} in .http files

## Done
- Added `-D` / `--define` flag (repeatable) for inline key=value variables
- Variables stored in `programmaticVars` with highest precedence (overrides .env, OS env, globals)
- Supports multiple `-D key=value -D another=val`
- Works with `--define` long form
- 231 tests pass
