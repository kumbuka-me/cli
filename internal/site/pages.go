package site

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	md "github.com/kumbuka-me/kumbuka/pkg/markdown"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// pageDiscovery accumulates Markdown pages and guards route uniqueness during a source walk.
type pageDiscovery struct {
	// sourceDir is the root currently being traversed.
	sourceDir string
	// pages contains discovered source pages.
	pages []sourcePage
	// routes maps generated routes to the source file that claimed them.
	routes map[string]string
}

// discoverPages discovers Markdown source files and maps them to static routes.
func discoverPages(sourceDir string) ([]sourcePage, error) {
	discovery := pageDiscovery{
		sourceDir: sourceDir,
		routes:    make(map[string]string),
	}

	if err := filepath.WalkDir(sourceDir, discovery.visit); err != nil {
		return nil, err
	}

	slices.SortFunc(discovery.pages, compareSourcePages)
	return discovery.pages, nil
}

// visit collects one Markdown file during source directory traversal.
func (d *pageDiscovery) visit(filename string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := rejectSymlink(filename, entry); err != nil {
		return err
	}
	if filename == d.sourceDir {
		return nil
	}
	if entry.IsDir() {
		if strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		return nil
	}
	if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
		return nil
	}

	relative, err := filepath.Rel(d.sourceDir, filename)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	route := markdownFileRoute(relative)
	if existing, found := d.routes[route]; found {
		return fmt.Errorf("markdown files %s and %s map to the same route %q", existing, relative, route)
	}

	d.routes[route] = relative
	title, hasTitle := markdownTitle(string(data), route)
	d.pages = append(d.pages, sourcePage{
		SourcePath:      relative,
		Route:           route,
		Title:           title,
		Markdown:        string(data),
		HasTitleHeading: hasTitle,
	})

	return nil
}

// compareSourcePages orders discovered pages by source path.
func compareSourcePages(left, right sourcePage) int {
	return cmp.Compare(left.SourcePath, right.SourcePath)
}

// hasHomePage reports whether the discovered pages contain the root route.
func hasHomePage(pages []sourcePage) bool {
	return slices.ContainsFunc(pages, func(page sourcePage) bool {
		return page.Route == ""
	})
}

// indexPages builds source-route and wiki-link lookup indexes.
func indexPages(pages []sourcePage) (map[string]string, map[string]string) {
	routesBySource := make(map[string]string, len(pages))
	wikiTargets := make(map[string]string, len(pages)*2)
	ambiguousWikiTargets := make(map[string]bool)

	for _, page := range pages {
		routesBySource[page.SourcePath] = page.Route
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Route), page.Route)
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Title), page.Route)
	}
	for target := range ambiguousWikiTargets {
		delete(wikiTargets, target)
	}

	return routesBySource, wikiTargets
}

// markdownFileRoute maps a Markdown source filename to its clean static route.
func markdownFileRoute(filename string) string {
	clean := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(filename)), "/")
	clean = strings.TrimSuffix(clean, path.Ext(clean))

	if path.Base(clean) == "index" {
		clean = path.Dir(clean)
		if clean == "." {
			clean = ""
		}
	}

	return strings.Trim(clean, "/")
}

// markdownFence tracks an open fenced code block while scanning page titles.
type markdownFence struct {
	// marker is the backtick or tilde character opening the fence.
	marker byte
	// length is the opening marker run length required by a closing fence.
	length int
}

// markdownTitle extracts the first Markdown level-one heading or derives a title from the route.
func markdownTitle(source, route string) (title string, hasTitle bool) {
	var fence markdownFence
	previous := ""
	previousCanBeSetext := false

	for line := range strings.SplitSeq(strings.TrimPrefix(source, "\ufeff"), "\n") {
		line = strings.TrimSuffix(line, "\r")
		marker, length, rest, isFence := parseMarkdownFence(line)
		if fence.marker != 0 {
			if closesMarkdownFence(fence, marker, length, rest, isFence) {
				fence = markdownFence{}
			}
			continue
		}
		if isFence {
			fence = markdownFence{marker: marker, length: length}
			previousCanBeSetext = false
			continue
		}

		if title, ok := markdownATXH1(line); ok {
			return title, true
		}
		if previousCanBeSetext && markdownSetextH1(line) {
			return strings.TrimSpace(previous), true
		}

		trimmed, contentLine := markdownContentLine(line)
		previous = trimmed
		previousCanBeSetext = contentLine && trimmed != ""
	}

	return derivedPageTitle(route), false
}

// parseMarkdownFence reports fenced-code markers indented by at most three spaces.
func parseMarkdownFence(line string) (marker byte, length int, rest string, ok bool) {
	content, valid := markdownIndentedLine(line)
	if !valid || len(content) < 3 || (content[0] != '`' && content[0] != '~') {
		return 0, 0, "", false
	}

	marker = content[0]
	for length < len(content) && content[length] == marker {
		length++
	}
	if length < 3 {
		return 0, 0, "", false
	}
	rest = content[length:]
	if marker == '`' && strings.ContainsRune(rest, '`') {
		return 0, 0, "", false
	}
	return marker, length, rest, true
}

// closesMarkdownFence reports whether a parsed marker closes the active fenced code block.
func closesMarkdownFence(fence markdownFence, marker byte, length int, rest string, parsed bool) bool {
	return parsed && marker == fence.marker && length >= fence.length && strings.TrimSpace(rest) == ""
}

// markdownATXH1 returns the text of a valid level-one ATX heading.
func markdownATXH1(line string) (string, bool) {
	content, valid := markdownIndentedLine(line)
	if !valid || len(content) == 0 || content[0] != '#' {
		return "", false
	}
	if len(content) > 1 && content[1] == '#' {
		return "", false
	}
	if len(content) > 1 && content[1] != ' ' && content[1] != '\t' {
		return "", false
	}

	title := strings.TrimSpace(strings.TrimPrefix(content, "#"))
	if title == "" {
		return "", false
	}
	return trimATXClosingHashes(title), true
}

// trimATXClosingHashes removes an optional whitespace-delimited closing hash sequence.
func trimATXClosingHashes(title string) string {
	end := len(title)
	for end > 0 && title[end-1] == '#' {
		end--
	}
	if end == len(title) || end == 0 || (title[end-1] != ' ' && title[end-1] != '\t') {
		return title
	}
	return strings.TrimSpace(title[:end])
}

// markdownSetextH1 reports whether line is a level-one Setext underline.
func markdownSetextH1(line string) bool {
	content, valid := markdownIndentedLine(line)
	if !valid {
		return false
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	for index := range len(content) {
		if content[index] != '=' {
			return false
		}
	}
	return true
}

// markdownContentLine returns text eligible to precede a Setext heading.
func markdownContentLine(line string) (string, bool) {
	content, valid := markdownIndentedLine(line)
	if !valid {
		return "", false
	}
	return strings.TrimSpace(content), true
}

// markdownIndentedLine strips up to three leading spaces and rejects indented code lines.
func markdownIndentedLine(line string) (string, bool) {
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
		if spaces == 4 {
			return "", false
		}
	}
	if spaces < len(line) && line[spaces] == '\t' {
		return "", false
	}
	return line[spaces:], true
}

// derivedPageTitle creates a readable fallback title from a static route.
func derivedPageTitle(route string) string {
	if route == "" {
		return "Home"
	}

	segment := path.Base(route)
	segment = strings.NewReplacer("-", " ", "_", " ").Replace(segment)
	words := strings.Fields(segment)
	for index, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// registerWikiTarget records an unambiguous wiki-link target.
func registerWikiTarget(targets map[string]string, ambiguous map[string]bool, target, route string) {
	target = strings.Trim(target, "/")
	if target == "" && route != "" {
		return
	}
	if existing, found := targets[target]; found && existing != route {
		ambiguous[target] = true
		return
	}

	targets[target] = route
}

// validateWikiLinks rejects source pages that reference unresolved wiki targets.
func validateWikiLinks(page sourcePage, targets map[string]string) error {
	for _, target := range md.Links(page.Markdown) {
		if _, found := targets[target]; !found {
			return fmt.Errorf("%s contains unresolved wiki link %q", page.SourcePath, target)
		}
	}

	return nil
}

// expandedPrefixes returns navigation ancestors that should be expanded for a route.
func expandedPrefixes(route string) []string {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) <= 1 {
		return nil
	}

	expanded := make([]string, 0, len(parts)-1)
	for index := 1; index < len(parts); index++ {
		expanded = append(expanded, strings.Join(parts[:index], "/"))
	}

	return expanded
}

// processRenderedHTML removes the duplicate title, rewrites local URLs, and extracts search text.
func processRenderedHTML(
	rendered, sourcePath string,
	removeTitle bool,
	routesBySource map[string]string,
	basePath string,
) (renderedHTML string, searchText string, err error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(rendered), contextNode)
	if err != nil {
		return "", "", err
	}

	if removeTitle {
		nodes = removeFirstHeading(nodes)
	}
	for _, node := range nodes {
		if err := rewriteHTMLURLs(node, sourcePath, routesBySource, basePath); err != nil {
			return "", "", err
		}
	}

	var htmlOutput bytes.Buffer
	for _, node := range nodes {
		if err := xhtml.Render(&htmlOutput, node); err != nil {
			return "", "", err
		}
	}

	return htmlOutput.String(), normalizeSearchText(textFromNodes(nodes)), nil
}

// removeFirstHeading removes the first top-level heading node from rendered fragments.
func removeFirstHeading(nodes []*xhtml.Node) []*xhtml.Node {
	for index, node := range nodes {
		if node.Type == xhtml.ElementNode && node.Data == "h1" {
			return slices.Delete(nodes, index, index+1)
		}
	}

	return nodes
}

// isRewritableURLAttribute reports whether a static-page attribute contains a navigable local URL.
func isRewritableURLAttribute(element, attribute string) bool {
	switch element {
	case "a":
		return attribute == "href"
	case "img":
		return attribute == "src"
	default:
		return false
	}
}

// rewriteHTMLURLs recursively rewrites navigable local URLs in rendered HTML.
func rewriteHTMLURLs(node *xhtml.Node, sourcePath string, routesBySource map[string]string, basePath string) error {
	if node.Type == xhtml.ElementNode {
		for index := range node.Attr {
			attribute := &node.Attr[index]
			if !isRewritableURLAttribute(node.Data, attribute.Key) {
				continue
			}

			rewritten, err := rewriteLocalURL(attribute.Val, sourcePath, routesBySource, basePath)
			if err != nil {
				return err
			}
			attribute.Val = rewritten
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := rewriteHTMLURLs(child, sourcePath, routesBySource, basePath); err != nil {
			return err
		}
	}

	return nil
}

// isRewritableLocalURL reports whether a parsed URL refers to a non-empty path inside the generated site.
func isRewritableLocalURL(value string, parsed *url.URL) bool {
	if parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	if strings.HasPrefix(value, "//") {
		return false
	}

	return parsed.Path != ""
}

// rewriteLocalURL rewrites one local Markdown or asset URL for the generated site.
func rewriteLocalURL(value, sourcePath string, routesBySource map[string]string, basePath string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return value, nil
	}

	basePath = ensureBasePath(basePath)
	if !strings.HasPrefix(value, "//") && strings.HasPrefix(value, basePath) {
		pathname, err := url.PathUnescape(value)
		if err != nil {
			return "", err
		}
		return (&url.URL{Path: pathname}).EscapedPath(), nil
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if !isRewritableLocalURL(value, parsed) {
		return value, nil
	}

	trailingSlash := strings.HasSuffix(parsed.Path, "/")
	resolved := resolveLocalPath(parsed.Path, sourcePath)
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("link %q escapes the documentation source", value)
	}

	if strings.EqualFold(path.Ext(resolved), ".md") {
		route, found := routesBySource[resolved]
		if !found {
			return "", fmt.Errorf("markdown link %q points to missing file %s", value, resolved)
		}
		parsed.Path = pageURL(basePath, route)
	} else {
		parsed.Path = basePath + strings.TrimPrefix(resolved, "/")
		if trailingSlash && !strings.HasSuffix(parsed.Path, "/") {
			parsed.Path += "/"
		}
	}

	return parsed.String(), nil
}

// resolveLocalPath resolves one URL path relative to its Markdown source file.
func resolveLocalPath(value, sourcePath string) string {
	if strings.HasPrefix(value, "/") {
		return strings.TrimPrefix(path.Clean(value), "/")
	}

	resolved := path.Clean(path.Join(path.Dir(sourcePath), value))
	if resolved == "." {
		return ""
	}

	return resolved
}

// textFromNodes extracts searchable text from rendered HTML nodes.
func textFromNodes(nodes []*xhtml.Node) string {
	var output strings.Builder
	for _, node := range nodes {
		appendNodeText(&output, node)
	}

	return output.String()
}

// appendNodeText recursively appends searchable text while skipping scripts and styles.
func appendNodeText(output *strings.Builder, node *xhtml.Node) {
	if node.Type == xhtml.TextNode {
		output.WriteString(node.Data)
		output.WriteByte(' ')
	}
	if node.Type == xhtml.ElementNode && (node.Data == "script" || node.Data == "style") {
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendNodeText(output, child)
	}
}

// normalizeSearchText collapses rendered text into a single whitespace-normalized string.
func normalizeSearchText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
