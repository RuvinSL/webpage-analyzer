package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type ConcurrentLinkChecker struct {
	httpClient     interfaces.HTTPClient
	workerPoolSize int
	logger         interfaces.Logger
	metrics        interfaces.MetricsCollector

	jobQueue    chan linkCheckJob
	resultQueue chan models.LinkStatus
	workerWG    sync.WaitGroup
	stopChan    chan struct{}
	started     bool         // fixed - Ruvin
	mu          sync.RWMutex // fixed - Ruvin
}

type linkCheckJob struct {
	ctx  context.Context
	link models.Link
}

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
		started:        false, // added fixed - Ruvin
	}
}

// fixed the code and added concurrent start
func (c *ConcurrentLinkChecker) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		c.logger.Warn("Link checker already started")
		return
	}

	c.logger.Info("Starting link checker worker pool", "workers", c.workerPoolSize)

	// Start workers
	for i := 0; i < c.workerPoolSize; i++ {
		c.workerWG.Add(1)
		go c.worker(ctx, i)
	}

	c.started = true
}

func (c *ConcurrentLinkChecker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	c.logger.Info("Stopping link checker worker pool")

	// Signal workers to stop
	close(c.stopChan)

	// Wait for all workers to finish
	c.workerWG.Wait()

	// Close channels
	close(c.jobQueue)
	close(c.resultQueue)

	c.started = false
	c.logger.Info("Link checker worker pool stopped")
}

// CheckLinks checks multiple links concurrently
func (c *ConcurrentLinkChecker) CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
	if len(links) == 0 {
		return []models.LinkStatus{}, nil
	}

	start := time.Now()
	c.logger.Info("Starting batch link check", "link_count", len(links))

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create dedicated channels for this batch to avoid interference
	batchJobQueue := make(chan linkCheckJob, len(links))
	batchResultQueue := make(chan models.LinkStatus, len(links))

	// Start workers for this batch
	var workerWG sync.WaitGroup
	for i := 0; i < c.workerPoolSize; i++ {
		workerWG.Add(1)
		go func(workerID int) {
			defer workerWG.Done()
			for job := range batchJobQueue {
				status := c.CheckLink(job.ctx, job.link)
				select {
				case batchResultQueue <- status:
				case <-checkCtx.Done():
					return
				}
			}
		}(i)
	}

	// fixed Submit all jobs
	go func() {
		defer close(batchJobQueue)
		for _, link := range links {
			select {
			case batchJobQueue <- linkCheckJob{ctx: checkCtx, link: link}:
			case <-checkCtx.Done():
				return
			}
		}
	}()

	// Collect results
	results := make([]models.LinkStatus, 0, len(links))
	resultMap := make(map[string]models.LinkStatus)

	// Start result collector
	go func() {
		defer close(batchResultQueue)
		workerWG.Wait() // Wait for all workers to finish before closing result channel
	}()

	// Collect all results
	for i := 0; i < len(links); i++ {
		select {
		case status := <-batchResultQueue:
			resultMap[status.Link.URL] = status
		case <-checkCtx.Done():
			c.logger.Warn("Context cancelled during result collection")
			break
		}
	}

	// Convert map to slice maintaining order
	for _, link := range links {
		if status, exists := resultMap[link.URL]; exists {
			results = append(results, status)
		} else {
			// Create timeout result for unchecked links
			results = append(results, models.LinkStatus{
				Link:       link,
				Accessible: false,
				StatusCode: 0,
				Error:      "Check timeout or not processed",
				CheckedAt:  time.Now(),
			})
		}
	}

	duration := time.Since(start)
	c.logger.Info("Batch link check completed",
		"link_count", len(links),
		"processed_count", len(results),
		"duration", duration,
		"avg_time_per_link", duration/time.Duration(len(links)),
	)

	return results, nil
}

func (c *ConcurrentLinkChecker) CheckLink(ctx context.Context, link models.Link) models.LinkStatus {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		c.metrics.RecordLinkCheck(true, duration)
	}()

	c.logger.Debug("Checking link", "url", link.URL, "type", link.Type)

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform HTTP GET request
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
			case <-c.stopChan:
				// Stop signal received
				return
			}
		}
	}
}
