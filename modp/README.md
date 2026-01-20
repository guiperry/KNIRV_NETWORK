# KNIRV Network P Model (`modp`)

This directory contains a formal model of the KNIRV Network, written in the P language. P is a state machine-based programming language for modeling and testing asynchronous, event-driven systems. This model is used to verify the correctness of the KNIRV Network's design and implementation by checking for deadlocks, race conditions, and other concurrency-related bugs.

## Project Structure

The `modp` directory is structured as a standard P project:

-   `KnirvNetwork.pproj`: The main project file that defines the project's name, input files, and output directory.
-   `types/`: Contains P files defining the data types used throughout the model.
    -   `network_types.p`: Defines common data structures and types for the entire network model.
-   `events/`: Contains P files defining the events (messages) that can be sent between state machines.
    -   `network_events.p`: Defines the set of events that represent interactions between the different network components.
-   `components/`: Contains the P state machine models for the various components of the KNIRV Network.
    -   `oracle/`: Models the KNIRVORACLE component and its internal state machines.
    -   `chain/`: Models the KNIRVCHAIN component.
    -   `graph/`: Models the KNIRVGRAPH component.
    -   `router/`: Models the KNIRVROUTER component.
    -   `nexus/`: Models the KNIRVNEXUS component.
    -   `base/`: Models the KNIRVBASE component.
-   `monitors/`: Contains P monitors that specify safety and liveness properties of the system.
    -   `network_invariants.p`: Defines the global invariants that must hold true for the network to be considered correct.
-   `tests/`: Contains the test scenarios for the model.
    -   `network_composition_tests.p`: Defines test cases that compose different components and run them against the monitors.
-   `output/`: The default directory where the output of the P model checker (`PChecker`) is generated.
-   `scripts/`: Contains helper scripts for building, testing, and managing the model.

## Getting Started

To build and test this model, you will need to have the P language toolchain installed.

### Building the Model

The model can be compiled using the `p compile` command. This will generate a C# or C representation of the model.

```bash
# Navigate to the modp directory
cd modp

# Compile the project
p compile
```

This command reads the `KnirvNetwork.pproj` file and compiles all the specified P files into a single executable or library in the `output` directory.

### Running Tests

Once the model is compiled, you can use the `p check` command to run the tests. This command systematically explores the state space of the model to find bugs.

```bash
# Run the test cases
p check -tc <test_case_name> -s <number_of_schedules>
```

-   `<test_case_name>` should be replaced with the name of a test case defined in `tests/network_composition_tests.p`.
-   `<number_of_schedules>` is the number of execution paths the model checker should explore.

## Modeling Approach

The KNIRV Network is modeled as a collection of communicating state machines. Each component of the network (e.g., `KNIRVORACLE`, `KNIRVCHAIN`) is represented by one or more P state machines. These machines communicate with each other by sending and receiving events, which are defined in the `events/` directory.

The `monitors/` directory contains special state machines called monitors, which observe the behavior of the system and check if it conforms to the specified safety and liveness properties. If a monitor detects a violation of an invariant, `PChecker` will report an error.

This approach allows us to formally verify the design of the KNIRV Network and ensure that it is free from a large class of concurrency bugs.
