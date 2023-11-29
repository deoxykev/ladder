package responsemodifers

import (
	"bytes"
	"encoding/json"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
	"io"
	"ladder/proxychain"
	//"strings"
)

// Outline creates an JSON representation of the article
func Outline() proxychain.ResponseModification {
	return func(chain *proxychain.ProxyChain) error {
		// Use readability
		opts := trafilatura.Options{
			IncludeImages: true,
			IncludeLinks:  true,
			//FavorPrecision:     true,
			FallbackCandidates: nil, // TODO: https://github.com/markusmobius/go-trafilatura/blob/main/examples/chained/main.go
			// implement fallbacks from	"github.com/markusmobius/go-domdistiller" and 	"github.com/go-shiori/go-readability"
			OriginalURL: chain.Request.URL,
		}

		result, err := trafilatura.Extract(chain.Response.Body, opts)
		if err != nil {
			return err
		}

		doc := createJSONDocument(result)
		jsonData, err := json.MarshalIndent(doc, "", "    ")
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer(jsonData)
		//doc := trafilatura.CreateReadableDocument(result)
		//reader := io.NopCloser(strings.NewReader(dom.OuterHTML(doc)))
		chain.Response.Body = io.NopCloser(buf)
		return nil
	}

}

// =======================================================================================
// credit @joncrangle https://github.com/everywall/ladder/issues/38#issuecomment-1831252934

type ImageContent struct {
	Type    string `json:"type"`
	URL     string `json:"url"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

type LinkContent struct {
	Type string `json:"type"`
	Href string `json:"href"`
	Data string `json:"data"`
}

type TextContent struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type JSONDocument struct {
	Success bool `json:"success"`
	Error   struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Cause   string `json:"cause"`
	} `json:"error"`
	Metadata struct {
		Title       string   `json:"title"`
		Author      string   `json:"author"`
		URL         string   `json:"url"`
		Hostname    string   `json:"hostname"`
		Description string   `json:"description"`
		Sitename    string   `json:"sitename"`
		Date        string   `json:"date"`
		Categories  []string `json:"categories"`
		Tags        []string `json:"tags"`
		License     string   `json:"license"`
	} `json:"metadata"`
	Content  []interface{} `json:"content"`
	Comments string        `json:"comments"`
}

func createJSONDocument(extract *trafilatura.ExtractResult) *JSONDocument {
	jsonDoc := &JSONDocument{}

	// Populate success
	jsonDoc.Success = true

	// Populate metadata
	jsonDoc.Metadata.Title = extract.Metadata.Title
	jsonDoc.Metadata.Author = extract.Metadata.Author
	jsonDoc.Metadata.URL = extract.Metadata.URL
	jsonDoc.Metadata.Hostname = extract.Metadata.Hostname
	jsonDoc.Metadata.Description = extract.Metadata.Description
	jsonDoc.Metadata.Sitename = extract.Metadata.Sitename
	jsonDoc.Metadata.Date = extract.Metadata.Date.Format("2006-01-02")
	jsonDoc.Metadata.Categories = extract.Metadata.Categories
	jsonDoc.Metadata.Tags = extract.Metadata.Tags
	jsonDoc.Metadata.License = extract.Metadata.License

	// Populate content
	if extract.ContentNode != nil {
		jsonDoc.Content = parseContent(extract.ContentNode)
	}

	// Populate comments
	if extract.CommentsNode != nil {
		jsonDoc.Comments = dom.OuterHTML(extract.CommentsNode)
	}

	return jsonDoc
}

func parseContent(node *html.Node) []interface{} {
	var content []interface{}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Data {
		case "img":
			image := ImageContent{
				Type:    "img",
				URL:     dom.GetAttribute(child, "src"),
				Alt:     dom.GetAttribute(child, "alt"),
				Caption: dom.GetAttribute(child, "caption"),
			}
			content = append(content, image)

		case "a":
			link := LinkContent{
				Type: "a",
				Href: dom.GetAttribute(child, "href"),
				Data: dom.InnerText(child),
			}
			content = append(content, link)

		case "h1":
			text := TextContent{
				Type: "h1",
				Data: dom.InnerText(child),
			}
			content = append(content, text)

		case "h2":
			text := TextContent{
				Type: "h2",
				Data: dom.InnerText(child),
			}
			content = append(content, text)

		case "h3":
			text := TextContent{
				Type: "h3",
				Data: dom.InnerText(child),
			}
			content = append(content, text)

		// continue with other tags

		default:
			text := TextContent{
				Type: "p",
				Data: dom.InnerText(child),
			}
			content = append(content, text)
		}
	}

	return content
}
