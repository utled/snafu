# TODO

- [x] set up traversal and metadata collection
- [x] set up file content reading
- [x] set up db initialization (sqlite)
- [] set up service initialization
- [x] write collected data to db
- [] strategy for more efficient DB write on initial scan
- [x] set up cli to manage program
- [x] set up orchestration
- [x] set up file system change monitoring
  - [x] decide sync and monitoring strategy
  - [] set up prioritization
  - [x] implement workflow
  - [] implement change logging
- [] set up better error handling and logging
- [x] implement basic search
- [x] implement TUI for search
- [] figure out and implement advanced search
- [] implement additional tagging
- [] figure out what else it's supposed to do
- [] figure out what to do in life

# BUG FIXES & CHANGES
## General
- [x] change time representations from combined Sec+Nsec to time.Time objects
- [] fix the multiplied creation of new directories
- [x] store content snippets without regex. only regex full content

## TUI
- [] try to get scrollable columns
- [x] interactive rows
- [x] implement "per keystroke"-search instead of on enter
- [x] implement preview pane
- [x] update db connection persistency