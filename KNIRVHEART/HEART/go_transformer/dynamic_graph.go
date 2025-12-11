package main

import (
	"fmt"
	"log"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Operation represents a recorded operation in the dynamic graph
type Operation struct {
	Type   string
	Inputs []*DynamicNode
	Output *DynamicNode
	OpFunc func(g *gorgonia.ExprGraph, inputs []*gorgonia.Node) *gorgonia.Node
}

// DynamicNode represents a node in the dynamic graph
type DynamicNode struct {
	Name       string
	Value      *gorgonia.Node
	IsParam    bool
	RequiresGrad bool
	Grad       *gorgonia.Node
	Operation  *Operation
}

// DynamicGraph wraps Gorgonia to provide dynamic graph functionality
type DynamicGraph struct {
	Graph      *gorgonia.ExprGraph
	Params     []*DynamicNode
	Constants  []*DynamicNode
	Operations []*Operation
	VM         gorgonia.VM
}

// NewDynamicGraph creates a new dynamic graph
func NewDynamicGraph() *DynamicGraph {
	g := gorgonia.NewGraph()
	return &DynamicGraph{
		Graph:      g,
		Params:     make([]*DynamicNode, 0),
		Operations: make([]*Operation, 0),
		VM:         nil, // Will be created in Forward()
	}
}

// Parameter creates a learnable parameter
func (dg *DynamicGraph) Parameter(name string, shape []int, init gorgonia.InitWFn) *DynamicNode {
	var node *gorgonia.Node
	if len(shape) == 1 {
		// Vector
		node = gorgonia.NewVector(dg.Graph, tensor.Float32,
			gorgonia.WithShape(shape[0]),
			gorgonia.WithName(name),
			gorgonia.WithInit(init),
		)
	} else {
		// Matrix
		node = gorgonia.NewMatrix(dg.Graph, tensor.Float32,
			gorgonia.WithShape(shape...),
			gorgonia.WithName(name),
			gorgonia.WithInit(init),
		)
	}

	dynNode := &DynamicNode{
		Name:        name,
		Value:       node,
		IsParam:     true,
		RequiresGrad: true,
	}

	dg.Params = append(dg.Params, dynNode)
	return dynNode
}

// Constant creates a constant node
func (dg *DynamicGraph) Constant(value interface{}, name string) *DynamicNode {
	var node *gorgonia.Node
	switch v := value.(type) {
	case float32:
		node = gorgonia.NewScalar(dg.Graph, tensor.Float32, gorgonia.WithValue(v), gorgonia.WithName(name))
	case float64:
		node = gorgonia.NewScalar(dg.Graph, tensor.Float32, gorgonia.WithValue(float32(v)), gorgonia.WithName(name))
	case []float32:
		t := tensor.New(tensor.WithBacking(v), tensor.WithShape(len(v)))
		node = gorgonia.NewConstant(t, gorgonia.WithName(name))
	case int:
		node = gorgonia.NewScalar(dg.Graph, tensor.Int, gorgonia.WithValue(v), gorgonia.WithName(name))
	default:
		log.Fatalf("Unsupported constant type: %T", value)
	}

	dynNode := &DynamicNode{
		Name:        name,
		Value:       node,
		IsParam:     false,
		RequiresGrad: false,
	}

	dg.Constants = append(dg.Constants, dynNode)
	return dynNode
}

// Add performs dynamic addition
func (dg *DynamicGraph) Add(a, b *DynamicNode, name string) *DynamicNode {
	output := &DynamicNode{
		Name:        name,
		RequiresGrad: a.RequiresGrad || b.RequiresGrad,
	}

	op := &Operation{
		Type:   "add",
		Inputs: []*DynamicNode{a, b},
		Output: output,
		OpFunc: func(g *gorgonia.ExprGraph, inputs []*gorgonia.Node) *gorgonia.Node {
			result, err := gorgonia.Add(inputs[0], inputs[1])
			if err != nil {
				log.Fatal(err)
			}
			return result
		},
	}

	output.Operation = op
	dg.Operations = append(dg.Operations, op)

	return output
}

// Mul performs dynamic multiplication
func (dg *DynamicGraph) Mul(a, b *DynamicNode, name string) *DynamicNode {
	output := &DynamicNode{
		Name:        name,
		RequiresGrad: a.RequiresGrad || b.RequiresGrad,
	}

	op := &Operation{
		Type:   "mul",
		Inputs: []*DynamicNode{a, b},
		Output: output,
		OpFunc: func(g *gorgonia.ExprGraph, inputs []*gorgonia.Node) *gorgonia.Node {
			result, err := gorgonia.Mul(inputs[0], inputs[1])
			if err != nil {
				log.Fatal(err)
			}
			return result
		},
	}

	output.Operation = op
	dg.Operations = append(dg.Operations, op)

	return output
}

// Tanh performs dynamic tanh activation
func (dg *DynamicGraph) Tanh(x *DynamicNode, name string) *DynamicNode {
	output := &DynamicNode{
		Name:        name,
		RequiresGrad: x.RequiresGrad,
	}

	op := &Operation{
		Type:   "tanh",
		Inputs: []*DynamicNode{x},
		Output: output,
		OpFunc: func(g *gorgonia.ExprGraph, inputs []*gorgonia.Node) *gorgonia.Node {
			result, err := gorgonia.Tanh(inputs[0])
			if err != nil {
				log.Fatal(err)
			}
			return result
		},
	}

	output.Operation = op
	dg.Operations = append(dg.Operations, op)

	return output
}

// Sum computes the sum of a tensor
func (dg *DynamicGraph) Sum(x *DynamicNode, name string) *DynamicNode {
	output := &DynamicNode{
		Name:        name,
		RequiresGrad: x.RequiresGrad,
	}

	op := &Operation{
		Type:   "sum",
		Inputs: []*DynamicNode{x},
		Output: output,
		OpFunc: func(g *gorgonia.ExprGraph, inputs []*gorgonia.Node) *gorgonia.Node {
			result, err := gorgonia.Sum(inputs[0])
			if err != nil {
				log.Fatal(err)
			}
			return result
		},
	}

	output.Operation = op
	dg.Operations = append(dg.Operations, op)

	return output
}

// Forward executes the dynamic graph forward pass
func (dg *DynamicGraph) Forward() {
	// Create a new graph for this forward pass
	newGraph := gorgonia.NewGraph()

	// Map dynamic nodes to gorgonia nodes in the new graph
	nodeMap := make(map[*DynamicNode]*gorgonia.Node)

	// Create parameters in the new graph
	for _, param := range dg.Params {
		var newNode *gorgonia.Node
		shape := param.Value.Shape()
		if len(shape) == 1 {
			newNode = gorgonia.NewVector(newGraph, tensor.Float32,
				gorgonia.WithShape(shape[0]),
				gorgonia.WithName(param.Name),
				gorgonia.WithValue(param.Value.Value()),
			)
		} else {
			newNode = gorgonia.NewMatrix(newGraph, tensor.Float32,
				gorgonia.WithShape(shape...),
				gorgonia.WithName(param.Name),
				gorgonia.WithValue(param.Value.Value()),
			)
		}
		nodeMap[param] = newNode
		param.Value = newNode
	}

	// Create constants in the new graph
	for _, constant := range dg.Constants {
		var newNode *gorgonia.Node
		if constant.Value.IsScalar() {
			newNode = gorgonia.NewConstant(constant.Value.Value(), gorgonia.WithName(constant.Name))
		} else {
			newNode = gorgonia.NewConstant(constant.Value.Value(), gorgonia.WithName(constant.Name))
		}
		nodeMap[constant] = newNode
		constant.Value = newNode
	}

	// Execute operations in order
	for _, op := range dg.Operations {
		inputs := make([]*gorgonia.Node, len(op.Inputs))
		for i, input := range op.Inputs {
			if node, exists := nodeMap[input]; exists {
				inputs[i] = node
			} else {
				log.Fatalf("Input node not found: %s", input.Name)
			}
		}

		output := op.OpFunc(newGraph, inputs)
		nodeMap[op.Output] = output
		op.Output.Value = output
	}

	// Update the graph
	dg.Graph = newGraph
	dg.VM = gorgonia.NewTapeMachine(newGraph, gorgonia.BindDualValues())
}

// Backward computes gradients by reconstructing the static graph
func (dg *DynamicGraph) Backward(loss *DynamicNode) []*gorgonia.Node {
	// First, ensure forward has been called
	if dg.Graph == nil {
		log.Fatal("Must call Forward() before Backward()")
	}

	// Compute gradients using Gorgonia's automatic differentiation
	grads, err := gorgonia.Grad(loss.Value, dg.getLearnableNodes()...)
	if err != nil {
		log.Fatal("Failed to compute gradients:", err)
	}

	// Run the VM to compute gradients
	if err := dg.VM.RunAll(); err != nil {
		log.Fatal("VM execution failed:", err)
	}

	// Store gradients in parameters
	for i, param := range dg.Params {
		if i < len(grads) {
			param.Grad = grads[i]
		}
	}

	dg.VM.Reset()
	return grads
}

// Step performs an optimizer step
func (dg *DynamicGraph) Step(solver gorgonia.Solver, grads []*gorgonia.Node) error {
	valueGrads := gorgonia.NodesToValueGrads(grads)

	if err := solver.Step(valueGrads); err != nil {
		return err
	}

	return nil
}

// Reset clears the operations for a new forward pass
func (dg *DynamicGraph) Reset() {
	dg.Operations = make([]*Operation, 0)
}

// getLearnableNodes returns all learnable gorgonia nodes
func (dg *DynamicGraph) getLearnableNodes() []*gorgonia.Node {
	nodes := make([]*gorgonia.Node, len(dg.Params))
	for i, param := range dg.Params {
		nodes[i] = param.Value
	}
	return nodes
}

// PrintGraph prints the current graph structure
func (dg *DynamicGraph) PrintGraph() {
	fmt.Println("Dynamic Graph Structure:")
	fmt.Printf("Parameters: %d\n", len(dg.Params))
	for i, param := range dg.Params {
		fmt.Printf("  %d: %s %v\n", i, param.Name, param.Value.Shape())
	}
	fmt.Printf("Operations: %d\n", len(dg.Operations))
	for i, op := range dg.Operations {
		fmt.Printf("  %d: %s -> %s\n", i, op.Type, op.Output.Name)
	}
}