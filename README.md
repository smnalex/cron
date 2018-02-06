## Pretty print cron expressions
### Usage
```bash

go run main.go -e '*/15 0 1,15 * 1-5 /usr/bin/find'

```

//NOTE: Using strings.Builder (available on tip)
```

### All supported expession can be seen in `parser/parser_test.go`
