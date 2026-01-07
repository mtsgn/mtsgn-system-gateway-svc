package server

import (
	"strings"
	"sync"

	"github.com/mtsgn/mtsgn-system-gateway-svc/pkg/utils"
)

type ServiceConfig struct {
	Name     string
	Target   string
	Methods  []string
	Priority int  // Higher number = higher priority
	SkipAuth bool // If true, skip authentication
}

type PriorityRouter struct {
	root *RouteNode
	mu   sync.RWMutex
}

type RouteNode struct {
	children   map[string]*RouteNode
	service    *ServiceConfig
	isWildcard bool // for * patterns
}

func NewPriorityRouter() *PriorityRouter {
	return &PriorityRouter{
		root: &RouteNode{children: make(map[string]*RouteNode)},
	}
}

func (r *PriorityRouter) AddRoute(path string, service *ServiceConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	segments := utils.SplitPath(path)
	service.Priority = len(segments) // Depth-based priority

	r.insert(r.root, segments, service, 0)
}

func (r *PriorityRouter) insert(node *RouteNode, segments []string, service *ServiceConfig, depth int) {
	if len(segments) == 0 {
		node.service = service
		return
	}

	segment := segments[0]
	isWildcard := strings.HasPrefix(segment, "*") || strings.HasPrefix(segment, ":")

	key := segment
	if isWildcard {
		key = "*" // Group wildcards together
	}

	child, exists := node.children[key]
	if !exists {
		child = &RouteNode{
			children:   make(map[string]*RouteNode),
			isWildcard: isWildcard,
		}
		node.children[key] = child
	}

	r.insert(child, segments[1:], service, depth+1)
}

// FindBestMatch finds the most specific (highest priority) route
func (r *PriorityRouter) FindBestMatch(path string) *ServiceConfig {
	segments := utils.SplitPath(path)
	candidates := r.collectCandidates(r.root, segments, 0, []*ServiceConfig{})

	if len(candidates) == 0 {
		return nil
	}

	// Return the candidate with highest priority (most specific)
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Priority > best.Priority {
			best = candidate
		}
	}
	return best
}

func (r *PriorityRouter) collectCandidates(node *RouteNode, segments []string, depth int, results []*ServiceConfig) []*ServiceConfig {
	if node.service != nil {
		results = append(results, node.service)
	}

	if len(segments) == 0 {
		return results
	}

	segment := segments[0]

	// Try exact match first
	if child, exists := node.children[segment]; exists {
		results = r.collectCandidates(child, segments[1:], depth+1, results)
	}

	// Try wildcard match
	if child, exists := node.children["*"]; exists {
		results = r.collectCandidates(child, segments[1:], depth+1, results)
	}

	return results
}
