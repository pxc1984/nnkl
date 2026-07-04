package worker

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// extractText extracts text from the given blob content based on file type.
func extractText(content []byte, fileType string) (string, error) {
	switch fileType {
	case "docx":
		return extractDOCX(content)
	case "pptx":
		return extractPPTX(content)
	default:
		return "", fmt.Errorf("unsupported file type for direct extraction: %s", fileType)
	}
}

// ---------- DOCX ----------

// docx document.xml bodies are in the namespace:
//
//	<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
//	  <w:body>
//	    <w:p>            paragraph
//	      <w:r>          run
//	        <w:t>text</w:t>
//	      </w:r>
//	    </w:p>
//	  </w:body>
//	</w:document>

type docxDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func extractDOCX(content []byte) (string, error) {
	zf, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}

	docData, err := readZipEntry(zf, "word/document.xml")
	if err != nil {
		return "", fmt.Errorf("read word/document.xml: %w", err)
	}

	// Strip namespace prefixes since Go's XML decoder handles them poorly
	// when they aren't explicitly mapped.
	clean := stripXMLNamespace(docData, "w")

	var doc docxDocument
	if err := xml.Unmarshal(clean, &doc); err != nil {
		return "", fmt.Errorf("parse word/document.xml: %w", err)
	}

	var buf strings.Builder
	for _, p := range doc.Body.Paragraphs {
		for _, r := range p.Runs {
			buf.WriteString(r.Text)
		}
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}

// ---------- PPTX ----------

// PPTX slides are in:
//
//	ppt/slides/slide1.xml, slide2.xml, …
//
// Namespace: http://schemas.openxmlformats.org/drawingml/2006/main (a:)
//            http://schemas.openxmlformats.org/presentationml/2006/main (p:)
//
// <p:sld>
//   <p:cSld>
//     <p:spTree>
//       <p:sp>
//         <p:txBody>
//           <a:p>
//             <a:r>
//               <a:t>text</a:t>
//             </a:r>
//           </a:p>
//         </p:txBody>
//       </p:sp>
//     </p:spTree>
//   </p:cSld>
// </p:sld>

type pptxSlide struct {
	XMLName xml.Name `xml:"sld"`
	CSld    pptxCSld `xml:"cSld"`
}

type pptxCSld struct {
	SpTree pptxSpTree `xml:"spTree"`
}

type pptxSpTree struct {
	Shapes []pptxShape `xml:"sp"`
}

type pptxShape struct {
	TxBody pptxTxBody `xml:"txBody"`
}

type pptxTxBody struct {
	Paragraphs []pptxParagraph `xml:"p"`
}

type pptxParagraph struct {
	Runs []pptxRun `xml:"r"`
}

type pptxRun struct {
	Text string `xml:"t"`
}

func extractPPTX(content []byte) (string, error) {
	zf, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("open pptx zip: %w", err)
	}

	// Collect all slide XML files sorted by slide number.
	var slides []string
	for _, f := range zf.File {
		if matched, _ := filepath.Match("ppt/slides/slide*.xml", f.Name); matched {
			slides = append(slides, f.Name)
		}
	}
	sort.Strings(slides)

	var buf strings.Builder
	for _, slidePath := range slides {
		data, err := readZipEntry(zf, slidePath)
		if err != nil {
			continue
		}
		clean := stripXMLNamespace(data, "p", "a")

		var slide pptxSlide
		if err := xml.Unmarshal(clean, &slide); err != nil {
			continue
		}
		buf.WriteString("<!-- slide: " + filepath.Base(slidePath) + " -->\n\n")
		for _, sp := range slide.CSld.SpTree.Shapes {
			for _, p := range sp.TxBody.Paragraphs {
				for _, r := range p.Runs {
					buf.WriteString(r.Text)
				}
				buf.WriteString("\n\n")
			}
		}
	}
	return buf.String(), nil
}

// ---------- helpers ----------

func readZipEntry(zf *zip.Reader, path string) ([]byte, error) {
	for _, f := range zf.File {
		if f.Name == path {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("entry %q not found", path)
}

// stripXMLNamespace removes XML namespace prefixes by turning
// <prefix:tag> into <tag> and </prefix:tag> into </tag>.
func stripXMLNamespace(data []byte, prefixes ...string) []byte {
	s := string(data)
	for _, p := range prefixes {
		s = strings.ReplaceAll(s, "<"+p+":", "<")
		s = strings.ReplaceAll(s, "</"+p+":", "</")
		s = strings.ReplaceAll(s, " xmlns:"+p+"=\"", " _xmlns_"+p+"=\"")
	}
	return []byte(s)
}
