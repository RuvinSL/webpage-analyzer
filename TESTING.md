## Testing

### Run All Tests
```
go test ./...
```

### Run Unit Tests Only
```
go test -short ./...
```

### Run Integration Tests Only
```
go test -run Integration ./...
```

### Run Tests for Specific Package
```
go test ./services/gateway/...
go test ./services/analyzer/...
go test ./services/link-checker/...
```

### Test specific package with verbose output
```
go test -v ./services/analyzer/core
```

### Test with race detection
```
go test -race ./...
```

### Run Tests with Coverage
```
go test -coverprofile coverage.out ./...
go tool cover "-html=coverage.out" > coverage.html
```
