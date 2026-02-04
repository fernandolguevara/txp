package panels

import (
	"github.com/rivo/tview"

	"txp/internal/config"
)

type Library struct {
	Nav           *tview.TreeView
	NavPanel      *tview.Flex
	NavFilter     *tview.InputField
	NavHeader     *tview.TextView
	ContentPanel  *tview.Flex
	ContentHeader *tview.TextView
	Content       *tview.List
	LibraryGroup  *tview.TreeNode
	ArtistsNode   *tview.TreeNode
	AlbumsNode    *tview.TreeNode
	FavoritesNode *tview.TreeNode
	TracksNode    *tview.TreeNode
	GenresNode    *tview.TreeNode
	YearsNode     *tview.TreeNode
}

type LibraryRef struct {
	Path  string
	Label string
}

type NavRef struct {
	Kind  string
	Value string
}

func NewLibrary(cfg config.Config) *Library {
	nav := tview.NewTreeView()
	nav.SetBorder(false)

	rootNode := tview.NewTreeNode("Library")
	artistsNode := categoryNode("Artists", "artist")
	albumsNode := categoryNode("Albums", "album")
	favoritesNode := categoryNode("Favorites", "favorites")
	tracksNode := categoryNode("Tracks", "tracks")
	genresNode := categoryNode("Genres", "genre")
	yearsNode := categoryNode("Years", "year")
	rootNode.AddChild(artistsNode)
	rootNode.AddChild(albumsNode)
	rootNode.AddChild(favoritesNode)
	rootNode.AddChild(tracksNode)
	rootNode.AddChild(genresNode)
	rootNode.AddChild(yearsNode)

	libGroup := tview.NewTreeNode("Libraries")
	for _, lib := range cfg.Libraries {
		selected := false
		for _, sel := range cfg.SelectedLibraries {
			if sel == lib {
				selected = true
				break
			}
		}
		node := NewLibraryNode(lib, selected)
		libGroup.AddChild(node)
	}
	rootNode.AddChild(libGroup)

	nav.SetRoot(rootNode).SetCurrentNode(rootNode)

	content := tview.NewList().ShowSecondaryText(false)
	content.SetBorder(false)

	contentHeader := tview.NewTextView().SetDynamicColors(true)
	contentHeader.SetWrap(false)

	contentPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	contentPanel.SetBorder(true).SetTitle("[ (2) Content #0 ]")
	contentPanel.AddItem(contentHeader, 1, 0, false)
	contentPanel.AddItem(content, 0, 1, true)

	navFilter := tview.NewInputField().SetLabel("Filter: ")
	navFilter.SetFieldWidth(0)

	navHeader := tview.NewTextView().SetDynamicColors(true)
	navHeader.SetWrap(false)

	navPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	navPanel.SetBorder(true).SetTitle("[ (1) Library explorer #0 ]")
	navPanel.AddItem(navFilter, 1, 0, false)
	navPanel.AddItem(navHeader, 1, 0, false)
	navPanel.AddItem(nav, 0, 1, true)

	return &Library{
		Nav:           nav,
		NavPanel:      navPanel,
		NavFilter:     navFilter,
		NavHeader:     navHeader,
		ContentPanel:  contentPanel,
		ContentHeader: contentHeader,
		Content:       content,
		LibraryGroup:  libGroup,
		ArtistsNode:   artistsNode,
		AlbumsNode:    albumsNode,
		FavoritesNode: favoritesNode,
		TracksNode:    tracksNode,
		GenresNode:    genresNode,
		YearsNode:     yearsNode,
	}
}

func NewLibraryNode(path string, selected bool) *tview.TreeNode {
	label := path
	node := tview.NewTreeNode(label)
	node.SetReference(&LibraryRef{Path: path, Label: label})
	UpdateLibraryNodeText(node, label, selected)
	return node
}

func UpdateLibraryNodeText(node *tview.TreeNode, label string, selected bool) {
	prefix := "[ ] "
	if selected {
		prefix = "[•] "
	}
	node.SetText(prefix + label)
}

func categoryNode(label string, kind string) *tview.TreeNode {
	node := tview.NewTreeNode(label)
	node.SetReference(&NavRef{Kind: kind})
	return node
}
