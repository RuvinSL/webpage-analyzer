package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// ConcurrentLinkChecker implements concurrent link checking with worker pool
// Follows Single Responsibility Principle: Only responsible for coordinating concurrent link checks
type ConcurrentLinkChecker struct {
	httpClient     interfaces.HTTPClient
	workerPoolSize int
	logger         interfaces.Logger
	metrics        interfaces.MetricsCollector

	// Worker pool management
	jobQueue    chan linkCheckJob
	resultQueue chan models.LinkStatus
	workerWG    sync.WaitGroup
	stopChan    chan struct{}
}

type linkCheckJob struct {
	ctx  context.Context
	link models.Link
}

// NewConcurrentLinkChecker creates a new concurrent link checker
func NewConcurrentLinkChecker(
	httpClient interfaces.HTTPClient,
	workerPoolSize int,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *ConcurrentLinkChecker {
	return &ConcurrentLinkChecker{
		httpClient:     httpClient,
		workerPoolSize: workerPoolSize,
		logger:         logger,
		metrics:        metrics,
		jobQueue:       make(chan linkCheckJob, workerPoolSize*2),
		resultQueue:    make(chan models.LinkStatus, workerPoolSize*2),
		stopChan:       make(chan struct{}),
	}
}

// Start initializes and starts the worker pool
func (c *ConcurrentLinkChecker) Start(ctx context.Context) {
	c.logger.Info("Starting link checker worker pool", "workers", c.workerPoolSize)

	// Start workers
	for i := 0; i < c.workerPoolSize; i++ {
		c.workerWG.Add(1)
		go c.worker(ctx, i)
	}
}

// Stop gracefully shuts down the worker pool
func (c *ConcurrentLinkChecker) Stop() {
	c.logger.Info("Stopping link checker worker pool")

	// Signal workers to stop
	close(c.stopChan)

	// Close job queue to prevent new jobs
	close(c.jobQueue)

	// Wait for all workers to finish
	c.workerWG.Wait()

	// Close result queue
	close(c.resultQueue)

	c.logger.Info("Link checker worker pool stopped")
}

// CheckLinks checks multiple links concurrently
func (c *ConcurrentLinkChecker) CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
	if len(links) == 0 {
		return []models.LinkStatus{}, nil
	}

	start := time.Now()
	c.logger.Info("Starting batch link check", "link_count", len(links))

	// Create a context with timeout for the entire batch
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Result collection
	results := make([]models.LinkStatus, 0, len(links))
	resultMap := make(map[string]models.LinkStatus)
	resultMutex := sync.Mutex{}

	// Start result collector
	resultWG := sync.WaitGroup{}
	resultWG.Add(1)
	go func() {
		defer resultWG.Done()
		for status := range c.resultQueue {
			resultMutex.Lock()
			resultMap[status.Link.URL] = status
			resultMutex.Unlock()
		}
	}()

	// Submit jobs
	jobWG := sync.WaitGroup{}
	for _, link := range links {
		select {
		case <-checkCtx.Done():
			c.logger.Warn("Context cancelled during job submission")
			break
		default:
			jobWG.Add(1)
			go func(l models.Link) {
				defer jobWG.Done()
				select {
				case c.jobQueue <- linkCheckJob{ctx: checkCtx, link: l}:
					// Job submitted successfully
				case <-checkCtx.Done():
					// Context cancelled, create timeout result
					c.resultQueue <- models.LinkStatus{
						Link:       l,
						Accessible: false,
						StatusCode: 0,
						Error:      "Check timeout",
						CheckedAt:  time.Now(),
					}
				}
			}(link)
		}
	}

	// Wait for all jobs to be submitted
	jobWG.Wait()

	// Wait a bit for results to be processed
	time.Sleep(100 * time.Millisecond)

	// Close result queue for this batch
	resultWG.Wait()

	// Convert map to slice maintaining order
	resultMutex.Lock()
	for _, link := range links {
		if status, exists := resultMap[link.URL]; exists {
			results = append(results, status)
		} else {
			// Link wasn't checked (shouldn't happen)
			results = append(results, models.LinkStatus{
				Link:       link,
				Accessible: false,
				StatusCode: 0,
				Error:      "Not checked",
				CheckedAt:  time.Now(),
			})
		}
	}
	resultMutex.Unlock()

	duration := time.Since(start)
	c.logger.Info("Batch link check completed",
		"link_count", len(links),
		"duration", duration,
		"avg_time_per_link", duration/time.Duration(len(links)),
	)

	return results, nil
}

// CheckLink checks a single link
func (c *ConcurrentLinkChecker) CheckLink(ctx context.Context, link models.Link) models.LinkStatus {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		c.metrics.RecordLinkCheck(true, duration)
	}()

	c.logger.Debug("Checking link", "url", link.URL, "type", link.Type)

	// Create timeout context for individual link check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform HTTP HEAD request first (faster)
	resp, err := c.httpClient.Get(checkCtx, link.URL)

	status := models.LinkStatus{
		Link:      link,
		CheckedAt: time.Now(),
	}

	if err != nil {
		status.Accessible = false
		status.Error = err.Error()
		c.logger.Debug("Link check failed", "url", link.URL, "error", err)
		c.metrics.RecordLinkCheck(false, time.Since(start).Seconds())
	} else {
		status.Accessible = resp.StatusCode >= 200 && resp.StatusCode < 400
		status.StatusCode = resp.StatusCode
		if !status.Accessible {
			status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		c.logger.Debug("Link check completed", "url", link.URL, "status", resp.StatusCode)
	}

	return status
}

// worker is a goroutine that processes link check jobs
func (c *ConcurrentLinkChecker) worker(ctx context.Context, id int) {
	defer c.workerWG.Done()

	c.logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("Worker stopping due to context cancellation", "worker_id", id)
			return
		case <-c.stopChan:
			c.logger.Debug("Worker stopping due to stop signal", "worker_id", id)
			return
		case job, ok := <-c.jobQueue:
			if !ok {
				c.logger.Debug("Worker stopping, job queue closed", "worker_id", id)
				return
			}

			// Process the job
			status := c.CheckLink(job.ctx, job.link)

			// Send result
			select {
			case c.resultQueue <- status:
				// Result sent successfully
			case <-ctx.Done():
				// Context cancelled while sending result
				return
			}
		}
	}
}
