
# # Development helpers
# dev-setup:
# 	@echo "Setting up development environment..."
# 	@go mod download
# 	@go install github.com/golang/mock/mockgen@v1.6.0
# 	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# # Monitoring
# monitor-cpu:
# 	@echo "Starting CPU profiling..."
# 	@go tool pprof http://localhost:8080/debug/pprof/profile

# monitor-heap:
# 	@echo "Starting heap profiling..."
# 	@go tool pprof http://localhost:8080/debug/pprof/heap