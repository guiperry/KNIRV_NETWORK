# Nexus Refactor Implementation Plan
This plan refactors KNIRVNEXUS into a Modular Monolith. This single-binary backend approach is robust, performant, and dramatically easier to develop, debug, and deploy than the current multi-process system.

Phase 0: Initial Setup and Cleanup
Before diving into the refactor, let's establish some ground rules and initial cleanup steps.

Refactor agent-server as a Module: As requested, the agent-server will be implemented as an internal module, not a separate application. Its package will export functions and types for agent management, which will be called directly by other parts of the application.


Phase 1: Consolidate into a Single Backend Binary & Centralize Control
The goal of this phase is to eliminate the multiple entrypoints and the fragile orchestrator, establishing nexus-server as the single point of control for the entire backend.

Establish a Single Entrypoint:

The backend/cmd/nexus-server/main.go file will become the sole entrypoint for the backend.
Action: Delete the following redundant files and directories:
backend/cmd/dve-manager/
backend/cmd/validation-core/
backend/main.go (the orchestrator)
Create a Unified Server Struct:

Inside nexus-server/main.go, we will define a central Server struct. This will be the heart of the application, managing all shared resources and services.
Action: Define the struct to hold instances of all components:
go
 Show full code block 
// in backend/cmd/nexus-server/main.go
type Server struct {
    config           *config.Config
    db               *database.BuntDBManager // A single, shared DB manager
    router           *mux.Router
    httpServer       *http.Server
    p2pManager       *p2p.DVEP2PManager

    // All services will be held here
    dveManager       *dvemanager.DVEManager
    validationCore   *validation.ValidationCore
    cdeService       *cde.CDEService
    dnsService       *dns.DynamicDNSService
    agentServer      *agentserver.AgentServer
    // ... other services
}

Centralize Configuration and Lifecycle:

The main() function in nexus-server will become the master conductor for the application.
Action:
Merge all configuration settings from the deleted main.go files into a single, comprehensive config.go struct and a corresponding config.yaml file.
The main() function will now be responsible for:
Loading the unified configuration using Viper.
Initializing the single BuntDB database connection.
Instantiating each service (e.g., dvemanager.NewDVEManager(...)), passing the shared config and DB connection as arguments.
Creating a single HTTP router and server.
Calling the RegisterRoutes method for each service to attach its endpoints to the main router.
Starting all background tasks (like the DNS service's monitoring loops).
Managing a graceful shutdown that correctly stops each component in the reverse order of startup.
Phase 2: Refactor Services and Co-locate API Handlers
This phase reshapes each service into a self-contained module and moves the API logic directly into the services that own it, eliminating the internal/api directory.

Refactor Existing Services into Modules:

The core principle is to remove all server and process management logic from the service packages. They should only contain business logic.
Action (dvemanager, validation, dns, cde):
Modify the New... constructor for each service. It should now only accept dependencies (like the DB manager and config) and initialize the service's internal state. It should not start any servers or goroutines.
For services with background tasks, like dns.DynamicDNSService, the Start() method will remain but will only start its internal tickers. The main nexus-server will be responsible for calling this Start() method.
Remove any HTTP server setup or ListenAndServe calls from within these packages.

Co-locate API Handlers:

This is a critical step for creating a clean, maintainable codebase. The API handlers currently in internal/api/ will be moved to their respective service packages.
Action:
For each service, create a handlers.go file (e.g., internal/services/cde/handlers.go).
Move the relevant handler functions from internal/api/handlers_cde.go into this new file, changing them to be methods on the service struct. For example:
go
// in internal/services/cde/handlers.go
package cde

import "net/http"

func (s *CDEService) HandleListEnvironments(w http.ResponseWriter, r *http.Request) {
    // ... logic ...
}
Change the signature of these methods to take an http.ResponseWriter and *http.Request instead of a context.Context.


Define Routes per Service:
Each service will have its own routes.go file where it defines its endpoints.
Action:
For each service, create a routes.go file (e.g., internal/services/cde/routes.go).
In this file, define the routes for the service and register them with the provided mux.Router.

  Each service package will now export a `RegisterRoutes(router *mux.Router)` method. This method will define all of the service's endpoints and attach its handler methods.

  For example:

      ```go
      // in internal/services/cde/routes.go
      package cde

      import "github.com/gorilla/mux"

      func (s *CDEService) RegisterRoutes(r *mux.Router) {
          r.HandleFunc("/cde/environments", s.HandleListEnvironments).Methods("GET")
          r.HandleFunc("/cde/environments", s.HandleCreateEnvironment).Methods("POST")
          // ... other CDE routes
      }
      ```

After moving all handlers, **delete the `internal/api` directory**.

Refactor Middleware:
The generic middleware functions (logging, recovery, CORS) can be moved to a new shared package, for example, internal/web/middleware.
The AuthMiddleware will also be moved there and initialized once in nexus-server/main.go, then applied to the protected routes.

Phase 3: Simplify the Wrapper and Build Process
This phase simplifies the final packaging of the application, removing the complex multi-binary embedding in favor of a much cleaner approach.

Simplify the Top-Level Wrapper (KNIRVNEXUS/main.go):

The current approach of embedding an orchestrator that embeds other binaries is overly complex. We will simplify this.
Action:
Modify the build process so that the entire Go backend compiles into a single executable file, e.g., nexus-backend.
The top-level KNIRVNEXUS/main.go will now only embed two things: the out directory (frontend) and the single nexus-backend binary.
Its main function will be simplified to:
Extract the nexus-backend binary to a temporary location.
Execute nexus-backend as a single, managed subprocess.
Start its own Gin server to serve the frontend and proxy all /api requests to the nexus-backend subprocess.
This preserves the single-executable goal for the end-user while dramatically simplifying the internal process management.

Update the Build Process (Makefile):

Action: Modify the Makefile to reflect the new, simpler build flow.
A target build-backend will now run go build -o bin/nexus-backend ./backend/cmd/nexus-server/.
The final build or binary target will embed this single nexus-backend into the final KNIRVNEXUS wrapper.
Phase 4: Final Cleanup and Documentation
The final phase ensures the project is clean, and the documentation accurately reflects the new, superior architecture.

Remove Dead Code:

Action: Perform a final sweep of the repository to delete all now-unused files and directories. This includes the old cmd folders for the individual services and the entire internal/api directory.
Update Documentation:

Action:
Update README.md to describe the new modular monolith architecture.
Update NEXUS_MERGE.md and any other architectural documents.
Provide clear, simple instructions on how to build the project (make build) and run the single backend binary for development (go run ./backend/cmd/nexus-server).

Conclusion:
This refined plan removes unnecessary complexity, consolidates the backend into a single, manageable binary, and places all API handlers directly within their owning services. It simplifies the overall architecture without sacrificing functionality or performance. The result is a system that is easier to understand, test, and maintain.

By executing this revised plan, KNIRVNEXUS will be transformed into a modern, maintainable, and performant application that is a pleasure to work on and robust in deployment.