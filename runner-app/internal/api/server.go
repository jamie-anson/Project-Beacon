package api

import (
	"os"

	"github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	wsHub "github.com/jamie-anson/project-beacon-runner/internal/websocket"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// APIServer holds the dependencies for API handlers
type APIServer struct {
	golemService *golem.Service
	executor     *golem.ExecutionEngine
	validator    *models.JobSpecValidator
	db           *db.DB
	jobsSvc      *service.JobsService
	jobsRepo     *store.JobsRepo
	execsRepo    *store.ExecutionsRepo
	ipfsRepo     *store.IPFSRepo
	ipfsClient   *ipfs.Client
	ipfsBundler  *ipfs.Bundler
	q            *queue.Client // for health checks
	wsHub        *wsHub.Hub
}

// NewAPIServer creates a new API server with dependencies
func NewAPIServer(database *db.DB) *APIServer {
	// Initialize Golem service
	apiKey := os.Getenv("GOLEM_API_KEY")
	if apiKey == "" {
		apiKey = "test-key" // Default for testing
	}
	
	network := os.Getenv("GOLEM_NETWORK")
	if network == "" {
		network = "testnet" // Default for testing
	}
	
	golemService := golem.NewService(apiKey, network)
	executor := golem.NewExecutionEngine(golemService)
	validator := models.NewJobSpecValidator()
	
	// Initialize jobs service/repo if DB is available
	var jobsSvc *service.JobsService
	var jobsRepo *store.JobsRepo
	var execsRepo *store.ExecutionsRepo
	var ipfsRepo *store.IPFSRepo
	var ipfsClient *ipfs.Client
	var ipfsBundler *ipfs.Bundler
	
	if database != nil && database.DB != nil {
		jobsSvc = service.NewJobsService(database.DB)
		jobsRepo = store.NewJobsRepo(database.DB)
		execsRepo = store.NewExecutionsRepo(database.DB)
		ipfsRepo = store.NewIPFSRepo(database.DB)
		
		// Initialize IPFS client and bundler
		ipfsNodeURL := os.Getenv("IPFS_NODE_URL")
		if ipfsNodeURL == "" {
			ipfsNodeURL = "localhost:5001"
		}
		
		ipfsGateway := os.Getenv("IPFS_GATEWAY")
		if ipfsGateway == "" {
			ipfsGateway = "http://localhost:8080"
		}
		
		ipfsConfig := ipfs.Config{
			NodeURL: ipfsNodeURL,
			Gateway: ipfsGateway,
		}
		
		ipfsClient = ipfs.NewClient(ipfsConfig)
		ipfsBundler = ipfs.NewBundler(ipfsClient, ipfsRepo)
	}
	
	// Initialize queue client for health checks
	var q *queue.Client
	if database != nil && database.DB != nil {
		q = queue.MustNewFromEnv()
	}
	
	// Initialize WebSocket hub
	hub := wsHub.NewHub()
	go hub.Run()
	
	return &APIServer{
		golemService: golemService,
		executor:     executor,
		validator:    validator,
		db:           database,
		jobsSvc:      jobsSvc,
		jobsRepo:     jobsRepo,
		execsRepo:    execsRepo,
		ipfsRepo:     ipfsRepo,
		ipfsClient:   ipfsClient,
		ipfsBundler:  ipfsBundler,
		q:            q,
		wsHub:        hub,
	}
}
