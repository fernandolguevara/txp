package panels

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/rivo/tview"

	"txp/internal/config"
)

func buildLibrariesGroup(cfg config.Config) (*tview.Flex, *tview.Flex, *tview.Flex, *tview.List, *tview.TreeView, *tview.TextView, *tview.TextView) {
	librariesList := tview.NewList().ShowSecondaryText(false)
	librariesList.SetBorder(false)
	for _, lib := range cfg.Libraries {
		librariesList.AddItem(lib, "", 0, nil)
	}
	librariesMsg := tview.NewTextView().SetText("d: remove  r: rescan  p: add path")

	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	leftColumn.SetBorder(true).SetTitle("[ Added library paths ]")
	leftColumn.AddItem(librariesList, 0, 1, true)
	leftColumn.AddItem(librariesMsg, 1, 0, false)

	rootPath := userHome()
	libraryTree := newDirectoryTree(rootPath)
	treeMsg := tview.NewTextView().SetText("Enter: expand  Backspace: up  Space: add")

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	rightColumn.SetBorder(true).SetTitle("[ Folder tree (add new) ]")
	rightColumn.AddItem(libraryTree, 0, 1, true)
	rightColumn.AddItem(treeMsg, 1, 0, false)

	librariesGrp := tview.NewFlex().SetDirection(tview.FlexRow)
	librariesGrp.SetBorder(true).SetTitle("[ (1) Libraries ]")

	columns := tview.NewFlex().SetDirection(tview.FlexColumn)
	columns.AddItem(leftColumn, 0, 1, true)
	columns.AddItem(rightColumn, 0, 1, false)

	librariesGrp.AddItem(columns, 0, 1, true)

	return librariesGrp, leftColumn, rightColumn, librariesList, libraryTree, librariesMsg, treeMsg
}

func BuildSettingsDirectoryTree(rootPath string) *tview.TreeView {
	return newDirectoryTree(rootPath)
}

func PopulateSettingsDirectoryChildren(node *tview.TreeNode) {
	addDirectoryChildren(node)
}

func userHome() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

func newDirectoryTree(rootPath string) *tview.TreeView {
	rootNode := tview.NewTreeNode(rootPath)
	rootNode.SetReference(rootPath)
	rootNode.SetExpanded(true)
	addDirectoryChildren(rootNode)

	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetBorder(false)
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		path, ok := node.GetReference().(string)
		if !ok || path == "" {
			return
		}
		if len(node.GetChildren()) == 0 {
			addDirectoryChildren(node)
		}
		node.SetExpanded(!node.IsExpanded())
	})

	return tree
}

func addDirectoryChildren(node *tview.TreeNode) {
	path, ok := node.GetReference().(string)
	if !ok || path == "" {
		return
	}

	node.ClearChildren()
	if parent := filepath.Dir(path); parent != path {
		up := tview.NewTreeNode("..").SetReference(parent)
		node.AddChild(up)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	dirs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		return
	}

	sort.Strings(dirs)
	for _, name := range dirs {
		childPath := filepath.Join(path, name)
		child := tview.NewTreeNode(name).SetReference(childPath)
		node.AddChild(child)
	}
}
