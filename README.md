## Project overview, prerequisites, technology used

### overview
I follow microservices-based application approach to built the app in Golang that analyzes web pages and provides detailed information about their structure and content. The application follows SOLID design principles and focus on architectural complexity

### Prerequisites
Go 1.24.5 
Docker 26.0+
Docker Compose 2.20+
Air 1.62.0 (optional - only for Live reload when development: https://github.com/air-verse/air/releases)

### Technology used
Back-end Services:
API Gateway: http://localhost:8080 - Main entry point for web UI and API routing
Analyzer Service: http://localhost:8081 - Core analysis logic
Link Checker Service: http://localhost:8082 - Concurrent link validation
Metrics Service: http://localhost:9090 - Prometheus metrics endpoint

Front-end:
Web UI: http://localhost:8080 - Simple HTML and CSS interface
Pure JavaScript for form handling for back-end-focused approach

DevOps & Infrastructure:
Monitoring: Prometheus
Logging: Structured logging with slog
the app development approach supporting for the followings as well
Container Registry: Docker Hub (no files included)
CI/CD: GitHub Actions (no files included)
Cloud Platform: AWS EC2 (no files included)


### Any external dependencies, how to install them if there are any
Core dependencies
github.com/gorilla/mux v1.8.1          # HTTP routing
github.com/prometheus/client_golang v1.18.0  # Metrics
golang.org/x/net v0.19.0               # HTML parsing
github.com/stretchr/testify v1.8.4    # Testing framework
github.com/golang/mock v1.6.0         # Mocking framework


### Setup instructions for installation to run the project
Local Development
Clone and setup:
```
git clone https://github.com/RuvinSL/webpage-analyzer.git
cd webpage-analyzer
```

#### Option 1: (Recommended)
Run with Docker Compose:
```
docker-compose up --build 
```

Note: you can see the logs inside: <root>/logs

#### Option 2:
Run services individually:

Terminal 1 - API Gateway
```
cd services/gateway
go run main.go
```

Terminal 2 - Analyzer Service
```
cd services/analyzer
go run main.go
```

Terminal 3 - Link Checker Service
```
cd services/link-checker
go run main.go
```

Note: each service log will be created same folder that you run go command

#### Option 3:
Run services with Live reload: (you should install air dependency: https://github.com/air-verse/air/releases)

Terminal 1 - API Gateway
```
go to your root folder and run:
air
```

Terminal 2 - Analyzer Service
```
cd services/analyzer
air
```

Terminal 3 - Link Checker Service
```
cd services/link-checker
air
```

Note: each service log will be created same folder that you run go command

#### Access the application:
Web UI: http://localhost:8080
API: POST http://localhost:8080/api/v1/analyze
Metrics: http://localhost:8080/metrics
Health: http://localhost:8080/health

additional services with ports to check API health 
Analyzer: http://localhost:8081/health
Link-checker: http://localhost:8082/health
Prometheus: http://localhost:9090/targets


### The usage of the App with main functionalities and their role in the Application
Web Page Analysis is the main functionality of the app:
Enter a URL in the web form
Click "Analyze" to process
View comprehensive results including HTML version, title, headings, and links

Authentication & Security
CORS middleware for API security
Input validation for URLs

Logging
Structured JSON logging with slog
Log levels: DEBUG, INFO, WARN, ERROR

Error Handling
error responses with HTTP status codes
Detailed error messages for debugging

Performance Monitoring
Concurrent link checking and worker pool (in docker-compose file link-checker service has the configuration for pool size: WORKER_POOL_SIZE )
Prometheus metrics for reference

### Challenges have been faced and the approaches took to overcome
#### Concurrent Link Checking
Challenge: Efficiently checking multiple links without overwhelming target servers
Solution: Worker pool pattern with configurable concurrency limits and rate limiting

#### HTML Version Detection
Challenge: Accurately detecting various HTML doctypes
Solution: using regex pattern and strings.Contains() to covering DOCTYPE

#### Testing Microservices
Challenge: Testing distributed components in isolation
Solution: Interface based design with mock implementations for unit tests

#### live reload the App
Challenge: live reload the app when changes are made to code
Solution: install and setup .air.toml inside each service folder. then use "air" to run the app

#### Identify testing coverage
Challenge: Identify testing coverage of the app
Solution: run test using Coverage commands and go race detection
```
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Add possible improvements to the project
#### Load balance
Introduce Traefik (https://doc.traefik.io/traefik/routing/overview/) for Docker based load balance for app traffic

#### Caching Layer
Add Redis for caching analysis results. this can can easily setup into docker-compose and map the cache server port with app

#### Message Queue
Introduce Apache Kafka for async processing

#### Enhanced Monitoring
Add Grafana dashboards

#### Scalability / Kubernetes hosting
Introduce enterprise container management platform like www.portainer.io 

#### Security Enhancements
Add OAuth2 authentication
Implement API key management
Add rate limiting per user based IP

#### Feature Additions
SEO analysis capabilities

### Testing
done unit testing and integration testing

Please see the [TESTING.md](https://github.com/RuvinSL/webpage-analyzer/blob/main/TESTING.md) for more details