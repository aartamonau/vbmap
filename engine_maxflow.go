package main

import (
	"fmt"
	"math/rand"
	"sort"
)

type MaxFlowRIGenerator struct {
	dotPath    string
	dotVerbose bool
}

func makeMaxFlowRIGenerator() *MaxFlowRIGenerator {
	return &MaxFlowRIGenerator{dotPath: "", dotVerbose: false}
}

func (_ MaxFlowRIGenerator) String() string {
	return "maxflow"
}

func (gen *MaxFlowRIGenerator) SetParams(params map[string]string) error {
	for k, v := range params {
		switch k {
		case "dot":
			gen.dotPath = v
		case "dotVerbose":
			gen.dotVerbose = true
		default:
			return fmt.Errorf("unsupported parameter '%s'", k)
		}
	}

	return nil
}

func (gen MaxFlowRIGenerator) Generate(params VbmapParams) (RI RI, err error) {
	g := buildFlowGraph(params)

	diag.Print("Constructed flow graph.\n")
	diag.Print(g.graphStats)

	feasible, _ := g.FindFeasibleFlow()

	if gen.dotPath != "" {
		err := g.Dot(gen.dotPath, gen.dotVerbose)
		if err != nil {
			diag.Printf("Couldn't create dot file %s: %s",
				gen.dotPath, err.Error())
		}
	}

	if !feasible {
		return nil, ErrorNoSolution
	}

	return graphToRI(g, params), nil
}

func buildFlowGraph(params VbmapParams) (g *Graph) {
	graphName := fmt.Sprintf("Flow graph for RI (%s)", params)
	g = NewGraph(graphName)

	tags := params.Tags.TagsList()
	tagsNodes := params.Tags.TagsNodesMap()

	maxReplicationsPerTag := 0
	if params.NumReplicas != 0 {
		maxReplicationsPerTag = params.NumSlaves / params.NumReplicas
	}

	nodes := params.Nodes()
	for _, nodeIx := range rand.Perm(len(nodes)) {
		node := nodes[nodeIx]
		nodeTag := params.Tags[node]
		nodeSrcV := NodeSourceVertex(node)
		nodeSinkV := NodeSinkVertex(node)

		g.AddEdge(Source, nodeSrcV, params.NumSlaves, params.NumSlaves)
		g.AddEdge(nodeSinkV, Sink, params.NumSlaves, 0)

		for _, tagIx := range rand.Perm(len(tags)) {
			tag := tags[tagIx]

			if tag == nodeTag {
				continue
			}

			tagNodesCount := len(tagsNodes[tag])
			tagCapacity := Min(tagNodesCount, maxReplicationsPerTag)

			tagV := TagVertex(tag)
			g.AddEdge(nodeSrcV, tagV, tagCapacity, 0)
		}
	}

	for _, tagIx := range rand.Perm(len(tags)) {
		tag := tags[tagIx]
		tagNodes := tagsNodes[tag]
		tagV := TagVertex(tag)

		for _, tagNode := range tagNodes {
			tagNodeV := NodeSinkVertex(tagNode)

			g.AddEdge(tagV, tagNodeV, params.NumSlaves, 0)
		}
	}

	return
}

type nodeCount struct {
	node  Node
	count int
}

type nodeCountSlice []nodeCount

func (a nodeCountSlice) Len() int           { return len(a) }
func (a nodeCountSlice) Less(i, j int) bool { return a[i].count > a[j].count }
func (a nodeCountSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func graphToRI(g *Graph, params VbmapParams) (ri RI) {
	ri = make([][]bool, params.NumNodes)

	for i := range params.Nodes() {
		ri[i] = make([]bool, params.NumNodes)
	}

	for _, tag := range params.Tags.TagsList() {
		tagV := TagVertex(tag)

		inRepsCounts := make(nodeCountSlice, 0)
		outRepsCounts := make(nodeCountSlice, 0)

		for _, edge := range g.EdgesFromVertex(tagV) {
			// edge to node sink vertex
			dstNode := Node(edge.Dst.(NodeSinkVertex))

			count := nodeCount{dstNode, edge.Flow()}
			inRepsCounts = append(inRepsCounts, count)
		}

		for _, edge := range g.EdgesToVertex(tagV) {
			srcNode := Node(edge.Src.(NodeSourceVertex))

			count := nodeCount{srcNode, edge.Flow()}
			outRepsCounts = append(outRepsCounts, count)
		}

		sort.Sort(outRepsCounts)

		slavesLeft := len(inRepsCounts)
		slaveIx := 0

		for _, pair := range outRepsCounts {
			count := pair.count
			srcNode := int(pair.node)

			for count > 0 {
				if slavesLeft == 0 {
					panic(fmt.Sprintf("Ran out of slaves "+
						"on tag %v", tag))
				}

				for inRepsCounts[slaveIx].count == 0 {
					slaveIx = (slaveIx + 1) % len(inRepsCounts)
				}

				dstNode := int(inRepsCounts[slaveIx].node)

				if ri[srcNode][dstNode] {
					panic(fmt.Sprintf("Forced to use the "+
						"same slave %d twice (tag %v)",
						dstNode, tag))
				}

				ri[srcNode][dstNode] = true
				count -= 1

				inRepsCounts[slaveIx].count -= 1
				if inRepsCounts[slaveIx].count == 0 {
					slavesLeft -= 1
				}

				slaveIx = (slaveIx + 1) % len(inRepsCounts)
			}
		}
	}

	return
}
