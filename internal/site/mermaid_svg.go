package site

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	mermaidCLIExpectedVersion = "12.0.0"
	mermaidRenderSettings     = `{"securityLevel":"strict","htmlLabels":false,"flowchart":{"htmlLabels":false}}`
	maxMermaidSVGBytes        = 1 << 20
)

// materializeMermaidSVGs replaces passive Mermaid browser blocks with static SVG images.
// Interactive blocks are left intact for the existing isolated browser renderer.
func materializeMermaidSVGs(ctx context.Context, markup, sourcePath, outputDir, basePath string) (string, error) {
	fragment := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(markup), fragment)
	if err != nil {
		return "", fmt.Errorf("parse Mermaid page HTML: %w", err)
	}
	for _, node := range nodes {
		fragment.AppendChild(node)
	}

	changed, err := materializeMermaidNodes(ctx, fragment, sourcePath, outputDir, basePath)
	if err != nil {
		return "", err
	}
	if !changed {
		return markup, nil
	}

	var rendered strings.Builder
	for child := fragment.FirstChild; child != nil; child = child.NextSibling {
		if err := xhtml.Render(&rendered, child); err != nil {
			return "", fmt.Errorf("write Mermaid page HTML: %w", err)
		}
	}
	return rendered.String(), nil
}

// materializeMermaidNodes walks rendered content and converts supported passive blocks.
func materializeMermaidNodes(ctx context.Context, parent *xhtml.Node, sourcePath, outputDir, basePath string) (bool, error) {
	changed := false
	for node := parent.FirstChild; node != nil; {
		next := node.NextSibling
		if isMermaidBrowserBlock(node) {
			source, ok := mermaidCodeSource(node)
			if ok && !mermaidIsInteractive(source) {
				key := mermaidSVGKey(source)
				if err := generateMermaidSVGPair(ctx, outputDir, key, source); err != nil {
					return false, fmt.Errorf("%s: pre-render Mermaid diagram %s: %w", sourcePath, key[:12], err)
				}
				static := staticMermaidNode(key, basePath, mermaidAlignment(source))
				parent.InsertBefore(static, node)
				parent.RemoveChild(node)
				changed = true
			}
		} else {
			converted, err := materializeMermaidNodes(ctx, node, sourcePath, outputDir, basePath)
			if err != nil {
				return false, err
			}
			changed = changed || converted
		}
		node = next
	}
	return changed, nil
}

// isMermaidBrowserBlock recognizes only blocks produced by the installed Mermaid plugin.
func isMermaidBrowserBlock(node *xhtml.Node) bool {
	if node.Type != xhtml.ElementNode || node.Data != "div" {
		return false
	}
	plugin, module := "", ""
	for _, attribute := range node.Attr {
		switch attribute.Key {
		case "data-kumbuka-plugin":
			plugin = attribute.Val
		case "data-kumbuka-module":
			module = attribute.Val
		}
	}
	return plugin == "me.kumbuka.mermaid" && module == "diagrams"
}

// mermaidCodeSource reads the direct pre/code source without interpreting HTML markup.
func mermaidCodeSource(node *xhtml.Node) (string, bool) {
	for pre := node.FirstChild; pre != nil; pre = pre.NextSibling {
		if pre.Type != xhtml.ElementNode || pre.Data != "pre" {
			continue
		}
		for code := pre.FirstChild; code != nil; code = code.NextSibling {
			if code.Type != xhtml.ElementNode || code.Data != "code" {
				continue
			}
			for _, attribute := range code.Attr {
				if attribute.Key == "class" && strings.Contains(" "+attribute.Val+" ", " language-mermaid ") {
					var source strings.Builder
					appendMermaidText(&source, code)
					return source.String(), true
				}
			}
		}
	}
	return "", false
}

// appendMermaidText concatenates code text without processing Mermaid syntax.
func appendMermaidText(output *strings.Builder, node *xhtml.Node) {
	if node.Type == xhtml.TextNode {
		output.WriteString(node.Data)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendMermaidText(output, child)
	}
}

// mermaidIsInteractive recognizes the editor-owned Mermaid presentation directive.
func mermaidIsInteractive(source string) bool {
	for line := range strings.SplitSeq(source, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "%% kumbuka-interactive: true") {
			return true
		}
	}
	return false
}

// mermaidAlignment returns the selected, bounded alignment preset.
func mermaidAlignment(source string) string {
	for line := range strings.SplitSeq(source, "\n") {
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "%% kumbuka-align: left":
			return "left"
		case "%% kumbuka-align: right":
			return "right"
		}
	}
	return "center"
}

// mermaidSVGKey fingerprints the source and fixed build renderer configuration.
func mermaidSVGKey(source string) string {
	value := mermaidCLIExpectedVersion + "\x00" + mermaidRenderSettings + "\x00" + source
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

// staticMermaidNode creates a non-executable pair of images for the active site theme.
func staticMermaidNode(key, basePath, alignment string) *xhtml.Node {
	block := &xhtml.Node{
		Type:     xhtml.ElementNode,
		DataAtom: atom.Div,
		Data:     "div",
		Attr: []xhtml.Attribute{
			{Key: "class", Val: "kumbuka-static-illustration"},
			{Key: "style", Val: "text-align: " + alignment + ";"},
		},
	}
	for _, variant := range []string{"light", "dark"} {
		image := &xhtml.Node{
			Type:     xhtml.ElementNode,
			DataAtom: atom.Img,
			Data:     "img",
			Attr: []xhtml.Attribute{
				{Key: "class", Val: "kumbuka-static-illustration__" + variant},
				{Key: "src", Val: publicURLPath(basePath, "assets/diagrams/"+key+"-"+variant+".svg")},
				{Key: "alt", Val: "Mermaid diagram"},
			},
		}
		block.AppendChild(image)
	}
	return block
}

// generateMermaidSVGPair writes light and dark diagram variants exactly once per build.
func generateMermaidSVGPair(ctx context.Context, outputDir, key, source string) error {
	directory := filepath.Join(outputDir, "assets", "diagrams")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	light := filepath.Join(directory, key+"-light.svg")
	dark := filepath.Join(directory, key+"-dark.svg")
	if validMermaidSVGFile(light) && validMermaidSVGFile(dark) {
		return nil
	}

	binary := strings.TrimSpace(os.Getenv("MERMAID_CLI"))
	if binary == "" {
		binary = "mmdc"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return errors.New("Mermaid CLI is required for static diagrams; install @mermaid-js/mermaid-cli@12.0.0 or set MERMAID_CLI")
	}
	version, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return fmt.Errorf("read Mermaid CLI version: %w", err)
	}
	if strings.TrimSpace(string(version)) != mermaidCLIExpectedVersion {
		return fmt.Errorf("Mermaid CLI version %q is unsupported; expected %s", strings.TrimSpace(string(version)), mermaidCLIExpectedVersion)
	}

	workDir, err := os.MkdirTemp("", "kumbuka-mermaid-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	input := filepath.Join(workDir, "diagram.mmd")
	config := filepath.Join(workDir, "mermaid.json")
	if err := os.WriteFile(input, []byte(source), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(config, []byte(mermaidRenderSettings), 0o600); err != nil {
		return err
	}

	for _, variant := range []struct {
		name     string
		filename string
	}{
		{name: "default", filename: light},
		{name: "dark", filename: dark},
	} {
		generated := filepath.Join(workDir, variant.name+".svg")
		command := exec.CommandContext(ctx, binary, "--input", input, "--output", generated, "--theme", variant.name, "--backgroundColor", "transparent", "--configFile", config)
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Mermaid CLI %s: %w: %s", variant.name, err, strings.TrimSpace(string(output)))
		}
		content, err := os.ReadFile(generated)
		if err != nil {
			return err
		}
		if err := validateMermaidSVG(content); err != nil {
			return fmt.Errorf("unsafe generated %s SVG: %w (use interactive mode for diagrams with clickable elements)", variant.name, err)
		}
		if err := os.WriteFile(variant.filename, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// validMermaidSVGFile reports whether a previously generated asset is usable.
func validMermaidSVGFile(filename string) bool {
	content, err := os.ReadFile(filename)
	return err == nil && validateMermaidSVG(content) == nil
}

// validateMermaidSVG rejects active elements, external resources, and oversized SVG output.
func validateMermaidSVG(data []byte) error {
	if len(data) == 0 || len(data) > maxMermaidSVGBytes {
		return errors.New("SVG outside supported size range")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	root, depth := false, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch element := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root || element.Name.Local != "svg" || element.Name.Space != "http://www.w3.org/2000/svg" {
					return errors.New("expected SVG root")
				}
				root = true
			}
			switch strings.ToLower(element.Name.Local) {
			case "script", "foreignobject", "iframe", "object", "embed", "image", "use", "a", "animate", "animatetransform", "animatemotion", "set", "audio", "video":
				return errors.New("active SVG element")
			}
			for _, attribute := range element.Attr {
				name := strings.ToLower(attribute.Name.Local)
				value := strings.ToLower(strings.TrimSpace(attribute.Value))
				if strings.HasPrefix(name, "on") || name == "href" || name == "src" || unsafeMermaidCSS(value) {
					return errors.New("unsafe SVG attribute")
				}
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.Directive, xml.ProcInst:
			return errors.New("SVG directives are not supported")
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(element)) != "" {
				return errors.New("unexpected SVG text")
			}
			if unsafeMermaidCSS(string(element)) {
				return errors.New("unsafe SVG CSS")
			}
		}
	}
	if !root || depth != 0 {
		return errors.New("incomplete SVG document")
	}
	return nil
}

// unsafeMermaidCSS rejects code and external resource references in SVG styles.
func unsafeMermaidCSS(source string) bool {
	value := strings.ToLower(strings.TrimSpace(source))
	for _, forbidden := range []string{"javascript:", "@import", "expression(", "-moz-binding", "url(http", "url(//", "url(data:"} {
		if strings.Contains(value, forbidden) {
			return true
		}
	}
	for tail := value; ; {
		index := strings.Index(tail, "url(")
		if index < 0 {
			break
		}
		tail = strings.TrimSpace(tail[index+4:])
		if strings.HasPrefix(tail, "'#") || strings.HasPrefix(tail, "\"#") {
			tail = tail[2:]
		} else if strings.HasPrefix(tail, "#") {
			tail = tail[1:]
		} else {
			return true
		}
		if end := strings.IndexByte(tail, ')'); end >= 0 {
			tail = tail[end+1:]
		} else {
			return true
		}
	}
	return false
}
