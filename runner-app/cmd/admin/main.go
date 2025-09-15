package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "stuck-jobs":
		handleStuckJobs()
	case "recovery":
		handleRecovery()
	case "stats":
		handleStats()
	case "requeue":
		handleRequeue()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Project Beacon Admin CLI Tool

Usage:
  admin <command> [options]

Commands:
  stuck-jobs [limit]      List jobs stuck in 'created' status (default limit: 50)
  recovery [--execute]    Recover stale jobs in 'processing' status
                         Use --execute to actually perform recovery (default: dry-run)
                         Use --threshold N to set stale threshold in minutes (default: 10)
  stats                   Show comprehensive system statistics
  requeue <job-id>        Requeue a specific job

Examples:
  # List first 20 stuck jobs
  admin stuck-jobs 20

  # Dry-run recovery check
  admin recovery

  # Execute recovery for jobs older than 15 minutes
  admin recovery --execute --threshold 15

  # Show system statistics
  admin stats

  # Requeue specific job
  admin requeue job-12345

Environment Variables:
  DATABASE_URL           PostgreSQL connection string (required)
`)
}

func getDatabase() *sql.DB {
	database, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := database.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	return database
}

func handleStuckJobs() {
	database := getDatabase()
	defer database.Close()

	limit := 50
	if len(os.Args) > 2 {
		if l, err := strconv.Atoi(os.Args[2]); err == nil {
			limit = l
		}
	}

	jobsRepo := store.NewJobsRepo(database)
	executionsRepo := store.NewExecutionsRepo(database)
	diffsRepo := store.NewDiffsRepo(database)
	outboxRepo := store.NewOutboxRepo(database)
	
	jobsService := service.NewJobsService(database)
	jobsService.JobsRepo = jobsRepo
	jobsService.ExecutionsRepo = executionsRepo
	jobsService.DiffsRepo = diffsRepo
	jobsService.Outbox = outboxRepo
	
	repairService := service.NewJobRepairService(jobsService)
	stats, err := repairService.GetStuckJobsStats(context.Background())
	if err != nil {
		log.Fatalf("Failed to get stuck jobs stats: %v", err)
	}

	createdCount := int(stats["created_jobs"].(float64))
	runningCount := int(stats["running_jobs"].(float64))
	
	fmt.Printf("📊 Stuck Jobs Summary:\n")
	fmt.Printf("   Created Status: %d jobs\n", createdCount)
	fmt.Printf("   Running Status: %d jobs\n", runningCount)
	fmt.Printf("   Total Stuck: %d jobs\n\n", createdCount+runningCount)

	if createdCount == 0 && runningCount == 0 {
		fmt.Println("✅ No stuck jobs found!")
		return
	}

	fmt.Printf("🔧 Found %d stuck jobs. Proceeding with recovery...\n\n", createdCount+runningCount)
	
	fmt.Printf("🔍 Stuck Jobs Details (limit: %d):\n", limit)
	
	// Get created jobs
	if createdCount > 0 {
		fmt.Println("\n📝 Jobs in 'created' status:")
		createdJobsList, err := jobsRepo.ListJobsByStatus(context.Background(), "created", limit)
		if err != nil {
			log.Printf("Error listing created jobs: %v", err)
		} else {
			for i, job := range createdJobsList {
				if i >= limit {
					break
				}
				age := time.Since(job.CreatedAt)
				fmt.Printf("   %s (age: %v)\n", job.ID, age.Round(time.Minute))
			}
		}
	}

	// Get running jobs
	if runningCount > 0 {
		fmt.Println("\n🏃 Jobs in 'running' status:")
		runningJobsList, err := jobsRepo.ListJobsByStatus(context.Background(), "running", limit)
		if err != nil {
			log.Printf("Error listing running jobs: %v", err)
		} else {
			for i, job := range runningJobsList {
				if i >= limit {
					break
				}
				age := time.Since(job.CreatedAt)
				fmt.Printf("   %s (age: %v)\n", job.ID, age.Round(time.Minute))
			}
		}
	}
}

func handleRecovery() {
	database := getDatabase()
	defer database.Close()

	execute := false
	thresholdMinutes := 10

	// Parse arguments
	for i, arg := range os.Args[2:] {
		switch arg {
		case "--execute":
			execute = true
		case "--threshold":
			if i+1 < len(os.Args[2:]) {
				if t, err := strconv.Atoi(os.Args[2:][i+1]); err == nil {
					thresholdMinutes = t
				}
			}
		}
	}

	threshold := time.Duration(thresholdMinutes) * time.Minute
	recoveryService := service.NewJobRecoveryService(database)

	fmt.Printf("🔄 Job Recovery Operation\n")
	fmt.Printf("   Mode: %s\n", map[bool]string{true: "EXECUTE", false: "DRY-RUN"}[execute])
	fmt.Printf("   Threshold: %v\n\n", threshold)

	if !execute {
		// Dry run - just count stale jobs
		count, err := recoveryService.GetStaleJobsCount(context.Background(), threshold)
		if err != nil {
			log.Fatalf("Failed to get stale jobs count: %v", err)
		}

		fmt.Printf("🔍 Found %d stale processing jobs (older than %v)\n", count, threshold)
		if count > 0 {
			fmt.Printf("💡 Run with --execute to recover these jobs\n")
		} else {
			fmt.Printf("✅ No stale jobs found!\n")
		}
		return
	}

	// Execute recovery
	fmt.Printf("⚡ Executing recovery...\n")
	err := recoveryService.RecoverStaleJobs(context.Background(), threshold)
	if err != nil {
		log.Fatalf("Recovery failed: %v", err)
	}

	fmt.Printf("✅ Recovery completed successfully!\n")
	fmt.Printf("📊 Recovery completed: %d jobs processed\n", 0)
}

func handleStats() {
	database := getDatabase()
	defer database.Close()

	ctx := context.Background()
	
	// Initialize services
	jobsRepo := store.NewJobsRepo(database)
	executionsRepo := store.NewExecutionsRepo(database)
	diffsRepo := store.NewDiffsRepo(database)
	outboxRepo := store.NewOutboxRepo(database)
	
	jobsService := service.NewJobsService(database)
	jobsService.JobsRepo = jobsRepo
	jobsService.ExecutionsRepo = executionsRepo
	jobsService.DiffsRepo = diffsRepo
	jobsService.Outbox = outboxRepo
	
	// Job statistics
	repairService := service.NewJobRepairService(jobsService)
	stuckStats, err := repairService.GetStuckJobsStats(ctx)
	if err != nil {
		log.Printf("Failed to get stuck jobs stats: %v", err)
	}

	// Recovery statistics
	recoveryService := service.NewJobRecoveryService(database)
	staleCount, err := recoveryService.GetStaleJobsCount(ctx, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to get stale jobs count: %v", err)
	}

	// Job status distribution
	
	statusCounts := make(map[string]int)
	statuses := []string{"created", "processing", "completed", "failed"}
	
	for _, status := range statuses {
		jobs, err := jobsRepo.ListJobsByStatus(ctx, status, 1000)
		if err != nil {
			log.Printf("Error counting %s jobs: %v", status, err)
			continue
		}
		statusCounts[status] = len(jobs)
	}

	fmt.Printf("📊 Project Beacon Job Statistics\n")
	fmt.Printf("================================\n\n")

	fmt.Printf("📈 Job Status Distribution:\n")
	total := 0
	for _, status := range statuses {
		count := statusCounts[status]
		total += count
		icon := map[string]string{
			"created":    "📝",
			"processing": "⚙️",
			"completed":  "✅",
			"failed":     "❌",
		}[status]
		fmt.Printf("   %s %-10s: %d jobs\n", icon, status, count)
	}
	fmt.Printf("   📊 Total:      %d jobs\n\n", total)

	createdStuck := int(stuckStats["created_jobs"].(float64))
	runningStuck := int(stuckStats["running_jobs"].(float64))
	
	fmt.Printf("📊 System Statistics:\n")
	fmt.Printf("   Stuck Jobs (Created): %d\n", createdStuck)
	fmt.Printf("   Stuck Jobs (Running): %d\n", runningStuck)
	fmt.Printf("   ⏰ Stale (processing): %d jobs\n\n", staleCount)

	healthScore := 100
	if createdStuck > 0 {
		healthScore -= 20
	}
	if runningStuck > 0 {
		healthScore -= 20
	}
	if staleCount > 0 {
		healthScore -= 30
	}
	if statusCounts["failed"] > statusCounts["completed"]/10 {
		healthScore -= 20
	}

	healthIcon := "✅"
	if healthScore < 80 {
		healthIcon = "⚠️"
	}
	if healthScore < 60 {
		healthIcon = "🚨"
	}

	fmt.Printf("🏥 System Health Score: %s %d/100\n", healthIcon, healthScore)
	
	if healthScore < 100 {
		fmt.Printf("\n💡 Recommendations:\n")
		if createdStuck > 0 || runningStuck > 0 {
			fmt.Printf("   • Run 'admin recovery --execute' to fix stuck jobs\n")
		}
		if staleCount > 0 {
			fmt.Printf("   • Check for processing jobs stuck >10 minutes\n")
		}
		if statusCounts["failed"] > statusCounts["completed"]/10 {
			fmt.Printf("   • Investigate high failure rate\n")
		}
	}
}

func handleRequeue() {
	if len(os.Args) < 3 {
		fmt.Println("Error: job ID required")
		fmt.Println("Usage: admin requeue <job-id>")
		os.Exit(1)
	}

	jobID := os.Args[2]
	database := getDatabase()
	defer database.Close()

	ctx := context.Background()
	jobsRepo := store.NewJobsRepo(database)
	executionsRepo := store.NewExecutionsRepo(database)
	diffsRepo := store.NewDiffsRepo(database)
	outboxRepo := store.NewOutboxRepo(database)
	
	jobsService := service.NewJobsService(database)
	jobsService.JobsRepo = jobsRepo
	jobsService.ExecutionsRepo = executionsRepo
	jobsService.DiffsRepo = diffsRepo
	jobsService.Outbox = outboxRepo

	fmt.Printf("🔄 Requeuing job: %s\n", jobID)

	err := jobsService.RepublishJob(ctx, jobID)
	if err != nil {
		log.Fatalf("Failed to requeue job: %v", err)
	}

	fmt.Printf("✅ Job %s successfully requeued!\n", jobID)
}
