package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateNode(t *testing.T) {
	expected := "Node{name:hostid123, [192.168.1.1, 10.0.0.1]}"
	node := CreateNode("name", "hostid123", []string{"192.168.1.1", "10.0.0.1"})
	actual := node.String()

	if actual != expected {
		t.Errorf("CreateNode() = %v, want %v", actual, expected)
	}
}

func TestNodeEquals(t *testing.T) {
	node := CreateNode("name", "hostid123", []string{"192.168.1.1", "10.0.0.1"})
	otherNode := CreateNode("name", "hostid123", []string{"192.168.1.2", "10.0.0.2"})

	assert.True(t, node.Equals(node))
	assert.False(t, node.Equals(otherNode))
}