package collector

import (
	"fmt"

	"github.com/acronis/go-cti"
	"github.com/acronis/go-cti/metadata"
)

// TrieNode represents a node in the trie with type, value, and children
type TrieNode struct {
	Type     string
	Value    metadata.Entity
	Children map[string]*TrieNode
}

// NewTrieNode creates a new TrieNode with an initialized children map
func NewTrieNode() *TrieNode {
	return &TrieNode{
		Children: make(map[string]*TrieNode),
	}
}

// GetChild retrieves a child node by type and value
func (n *TrieNode) GetChild(childValue string) *TrieNode {
	key := childValue
	return n.Children[key]
}

// AddChild adds a child node if it doesn't exist and returns it
func (n *TrieNode) AddChild(childType, childValue string, object metadata.Entity) *TrieNode {
	key := childValue
	if child, exists := n.Children[key]; exists {
		return child
	}
	childNode := &TrieNode{
		Type:     childType,
		Value:    object,
		Children: make(map[string]*TrieNode),
	}
	n.Children[key] = childNode
	return childNode
}

func InsertIdentifier(root *TrieNode, expr *cti.Expression, object metadata.Entity) error {
	currentNode := root

	n := expr.Head
	// This will traverse the expression and create empty nodes or return existing ones for each part
	for n != nil {
		// TODO: Optimize
		nameNode := currentNode.AddChild("name", string(n.Vendor)+string(n.Package)+string(n.EntityName), nil)
		versionNode := nameNode.AddChild("version", n.Version.String(), nil)
		currentNode = versionNode
		n = n.Child
	}

	currentNode.Value = object

	return nil
}

func FindNode(root *TrieNode, expr *cti.Expression) (*TrieNode, error) {
	currentNode := root

	n := expr.Head
	for n != nil {
		// TODO: Optimize
		k := string(n.Vendor) + string(n.Package) + string(n.EntityName)
		nameNode := currentNode.GetChild(k)
		if nameNode == nil {
			return nil, fmt.Errorf("name node not found for %s", k)
		}
		currentNode = nameNode
		// Break if we reach the last node in the expression so we can observe the version node
		if n.Child == nil {
			break
		}

		versionNode := nameNode.GetChild(n.Version.String())
		if versionNode == nil {
			return nil, fmt.Errorf("version node not found for %s", n.Version.String())
		}
		currentNode = versionNode
		n = n.Child
	}

	return currentNode, nil
}
